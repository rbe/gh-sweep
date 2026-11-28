package project

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"time"
)

// Table is a report as data: one row for each line of the text report
// before the skipped lines, the skipped repositories, and the total.
// Each row has one value for each column, in the order of Columns.
type Table struct {
	Report       string
	Columns      []string
	Rows         [][]any
	Skipped      []Skipped
	TotalColumns []string
	Total        []any
}

// WriteJSON writes the table as one JSON object with the keys report, rows,
// skipped, and total, in that order, indented by two spaces.
func (t Table) WriteJSON(w io.Writer) error {
	var raw bytes.Buffer
	raw.WriteString(`{"report":`)
	if err := writeJSONValue(&raw, t.Report); err != nil {
		return err
	}
	raw.WriteString(`,"rows":[`)
	for i, row := range t.Rows {
		if i > 0 {
			raw.WriteByte(',')
		}
		if err := writeJSONObject(&raw, t.Columns, row); err != nil {
			return err
		}
	}
	raw.WriteString(`],"skipped":[`)
	for i, s := range t.Skipped {
		if i > 0 {
			raw.WriteByte(',')
		}
		if err := writeJSONObject(&raw, []string{"repository", "error"}, []any{s.NWO, s.Reason}); err != nil {
			return err
		}
	}
	raw.WriteString(`],"total":`)
	if err := writeJSONObject(&raw, t.TotalColumns, t.Total); err != nil {
		return err
	}
	raw.WriteByte('}')
	var out bytes.Buffer
	if err := json.Indent(&out, raw.Bytes(), "", "  "); err != nil {
		return err
	}
	out.WriteByte('\n')
	_, err := w.Write(out.Bytes())
	return err
}

// writeJSONObject writes one object whose keys stand in the order given.
func writeJSONObject(b *bytes.Buffer, keys []string, values []any) error {
	if len(keys) != len(values) {
		return fmt.Errorf("a row has %d values for %d columns", len(values), len(keys))
	}
	b.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		if err := writeJSONValue(b, key); err != nil {
			return err
		}
		b.WriteByte(':')
		if err := writeJSONValue(b, jsonValue(values[i])); err != nil {
			return err
		}
	}
	b.WriteByte('}')
	return nil
}

func writeJSONValue(b *bytes.Buffer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b.Write(data)
	return nil
}

// jsonValue turns a time into an RFC 3339 time in UTC, or nil when it is
// zero. Every other value stays as it is.
func jsonValue(v any) any {
	if t, ok := v.(time.Time); ok {
		if t.IsZero() {
			return nil
		}
		return t.UTC().Format(time.RFC3339)
	}
	return v
}

// WriteCSV writes a header record with the columns, result, and error, then
// one record for each row with result ok, then one record for each skipped
// repository with result skipped. It writes no total record.
func (t Table) WriteCSV(w io.Writer) error {
	out := csv.NewWriter(w)
	header := append(append([]string{}, t.Columns...), "result", "error")
	if err := out.Write(header); err != nil {
		return err
	}
	for _, row := range t.Rows {
		record := make([]string, 0, len(header))
		for _, v := range row {
			record = append(record, csvValue(v))
		}
		if err := out.Write(append(record, "ok", "")); err != nil {
			return err
		}
	}
	for _, s := range t.Skipped {
		record := make([]string, len(header))
		record[0] = s.NWO
		record[len(header)-2] = "skipped"
		record[len(header)-1] = s.Reason
		if err := out.Write(record); err != nil {
			return err
		}
	}
	out.Flush()
	return out.Error()
}

// csvValue is the text of one value in a CSV record. A zero time is empty.
func csvValue(v any) string {
	switch x := jsonValue(v).(type) {
	case nil:
		return ""
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}

// SummaryTable is the summary as data, with the same rows as SummaryLines.
func SummaryTable(inventories []Project) Table {
	t := Table{Report: "summary", Columns: []string{"repository", "artifacts", "caches", "bytes"},
		TotalColumns: []string{"artifacts", "caches", "bytes"}, Rows: [][]any{}}
	var artifacts, caches int
	var size int64
	for _, p := range inventories {
		if p.ArtN == 0 && p.CacheN == 0 {
			continue
		}
		t.Rows = append(t.Rows, []any{p.NWO, p.ArtN, p.CacheN, p.ArtBytes + p.CacheBytes})
		artifacts += p.ArtN
		caches += p.CacheN
		size += p.ArtBytes + p.CacheBytes
	}
	t.Total = []any{artifacts, caches, size}
	return t
}

// UsageTable is the storage usage breakdown as data, with the same rows as
// UsageLines, largest first.
func UsageTable(inventories []Project) Table {
	return TopUsageTable(inventories, 0)
}

// TopUsageTable is UsageTable with the rows of TopUsageLines.
func TopUsageTable(inventories []Project, top int) Table {
	t := Table{Report: "usage",
		Columns:      []string{"repository", "bytes", "artifact_bytes", "expired_artifact_bytes", "cache_bytes", "share_percent"},
		TotalColumns: []string{"bytes", "artifact_bytes", "expired_artifact_bytes", "cache_bytes"}, Rows: [][]any{}}
	used, all := usageOf(inventories, top)
	for _, u := range used {
		t.Rows = append(t.Rows, []any{u.name, u.total(), u.artifacts, u.expired, u.caches, sharePercent(u.total(), all.total())})
	}
	t.Total = []any{all.total(), all.artifacts, all.expired, all.caches}
	return t
}

// sharePercent is part as a percentage of all, rounded to one decimal.
func sharePercent(part, all int64) float64 {
	if all <= 0 {
		return 0
	}
	return math.Round(float64(part)*1000/float64(all)) / 10
}

// ArtifactTable is the artifact list as data, with the same rows as
// ArtifactLines.
func ArtifactTable(inventories []Project) Table {
	t := Table{Report: "artifact-list",
		Columns:      []string{"repository", "name", "id", "workflow", "bytes", "created", "expires", "status", "large"},
		TotalColumns: []string{"artifacts", "bytes", "expired", "expired_bytes"}, Rows: [][]any{}}
	var count, expired int
	var size, expiredSize int64
	for _, p := range inventories {
		for _, it := range artifactsBySize(p.Items) {
			t.Rows = append(t.Rows, []any{p.NWO, it.Name, it.ID, workflowOf(it), it.Size, it.When, it.Expires, artifactStatus(it), it.Large()})
			count++
			size += it.Size
			if it.Expired {
				expired++
				expiredSize += it.Size
			}
		}
	}
	t.Total = []any{count, size, expired, expiredSize}
	return t
}

// workflowOf is the workflow of an artifact in JSON and CSV: its name, or
// nil when the tool did not read it or GitHub gave none.
func workflowOf(it Item) any {
	if it.Workflow == "" || it.Workflow == "-" {
		return nil
	}
	return it.Workflow
}
