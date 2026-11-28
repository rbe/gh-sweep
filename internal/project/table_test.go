// id: ADR-0017

package project

import (
	"bytes"
	"errors"
	"testing"
)

func TestTableRefusesARowWithMoreValuesThanColumns(t *testing.T) {
	table := Table{Report: "summary", Columns: []string{"repository"}, Rows: [][]any{{"acme/widgets", 2}},
		TotalColumns: []string{}, Total: []any{}}
	var out bytes.Buffer
	if err := table.WriteJSON(&out); err == nil || out.Len() != 0 {
		t.Fatalf("err %v\nout\n%s", err, out.String())
	}
}

func TestUsageTableGivesTheShareWithOneDecimalAndZeroForNoStorage(t *testing.T) {
	inventories := []Project{
		{NWO: "acme/big", ArtN: 1, ArtBytes: 2},
		{NWO: "acme/small", ArtN: 1, ArtBytes: 1},
	}
	rows := UsageTable(inventories).Rows
	if rows[0][5] != 66.7 || rows[1][5] != 33.3 {
		t.Fatalf("rows %v", rows)
	}
	if sharePercent(1, 0) != 0 {
		t.Fatal("share of no storage is not 0")
	}
}

// fullDisk is a writer that refuses every write, as a full disk does.
type fullDisk struct{}

func (fullDisk) Write([]byte) (int, error) { return 0, errors.New("no space left on device") }

func TestTableReturnsTheErrorOfAWriterThatFails(t *testing.T) {
	table := SummaryTable([]Project{{NWO: "acme/widgets", ArtN: 1, ArtBytes: 5}})
	if err := table.WriteJSON(fullDisk{}); err == nil {
		t.Fatal("WriteJSON gave no error")
	}
	if err := table.WriteCSV(fullDisk{}); err == nil {
		t.Fatal("WriteCSV gave no error")
	}
}
