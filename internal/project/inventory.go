package project

import (
	"sort"
	"strings"
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

// Item is one artifact or one whole cache entry.
type Item struct {
	Kind    string
	ID      string
	Name    string
	Size    int64
	When    time.Time
	Meta    string
	Expired bool
	// Expires is the time GitHub deletes an artifact. It is zero for a cache.
	Expires time.Time
	Run     string
	On      bool
	Dead    bool
	Group   int
}

// Group is every artifact of one name, or every cache entry of one key.
type Group struct {
	Kind  string
	Name  string
	Start int
	Len   int
	Open  bool
	Bytes int64
}

func (it Item) searchBlob() string {
	b := strings.ToLower(it.Name + " " + it.Meta + " " + it.ID + " " + it.Kind)
	if it.Expired {
		b += " expired"
	}
	if it.Run != "" && it.Run != "0" {
		b += " " + strings.ToLower(it.Run)
	}
	return b
}

func (it Item) Matches(filter string) bool {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return true
	}
	return strings.Contains(it.searchBlob(), filter)
}

func (p *Project) Recompute() {
	p.ArtN, p.ExpN, p.CacheN = 0, 0, 0
	p.ArtBytes, p.ExpBytes, p.CacheBytes = 0, 0, 0
	for _, it := range p.Items {
		if it.Dead {
			continue
		}
		if it.Kind == "artifact" {
			p.ArtN++
			p.ArtBytes += it.Size
			if it.Expired {
				p.ExpN++
				p.ExpBytes += it.Size
			}
		} else {
			p.CacheN++
			p.CacheBytes += it.Size
		}
	}
	p.Scanned = true
}

// rebuildGroups sorts artifacts and caches into name/key groups, largest
// group first. Groups with ten or fewer items start expanded.
func (p *Project) RebuildGroups() {
	open := map[string]bool{}
	for _, g := range p.Groups {
		if g.Open {
			open[g.Kind+"\x00"+g.Name] = true
		}
	}
	arts := filterKind(p.Items, "artifact")
	caches := filterKind(p.Items, "cache")
	expand := len(p.Items) > 0 && len(p.Items) <= 10
	var items []Item
	var groups []Group
	items, groups = appendKind(items, groups, arts, "artifact", expand, open)
	items, groups = appendKind(items, groups, caches, "cache", expand, open)
	p.Items = items
	p.Groups = groups
	p.ArtGroups = nil
	p.CacheGroups = nil
	for i, g := range groups {
		if g.Kind == "artifact" {
			p.ArtGroups = append(p.ArtGroups, i)
		} else {
			p.CacheGroups = append(p.CacheGroups, i)
		}
	}
	order := func(idx []int) {
		sort.SliceStable(idx, func(a, b int) bool {
			ga, gb := groups[idx[a]], groups[idx[b]]
			if ga.Bytes != gb.Bytes {
				return ga.Bytes > gb.Bytes
			}
			return ga.Name < gb.Name
		})
	}
	order(p.ArtGroups)
	order(p.CacheGroups)
	p.Recompute()
}

func filterKind(items []Item, kind string) []Item {
	var out []Item
	for _, it := range items {
		if it.Kind == kind && !it.Dead {
			out = append(out, it)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		if !out[i].When.Equal(out[j].When) {
			return out[i].When.After(out[j].When)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func appendKind(items []Item, groups []Group, src []Item, kind string, expand bool, open map[string]bool) ([]Item, []Group) {
	for i := 0; i < len(src); {
		j := i + 1
		for j < len(src) && src[j].Name == src[i].Name {
			j++
		}
		g := Group{
			Kind:  kind,
			Name:  src[i].Name,
			Start: len(items),
			Len:   j - i,
			Open:  expand || open[kind+"\x00"+src[i].Name],
		}
		gi := len(groups)
		for k := i; k < j; k++ {
			src[k].Group = gi
			g.Bytes += src[k].Size
			items = append(items, src[k])
		}
		groups = append(groups, g)
		i = j
	}
	return items, groups
}

type RowKind int

const (
	RowSection RowKind = iota
	RowGroup
	RowItem
)

type ViewRow struct {
	Kind RowKind
	Ref  int
}

func (p *Project) Rows(filter string, wantArt, wantCache bool) []ViewRow {
	var rows []ViewRow
	if wantArt && p.sectionVisible("artifact", filter) {
		rows = append(rows, ViewRow{Kind: RowSection, Ref: 0})
		rows = append(rows, p.groupRows(p.ArtGroups, filter)...)
	}
	if wantCache && p.sectionVisible("cache", filter) {
		rows = append(rows, ViewRow{Kind: RowSection, Ref: 1})
		rows = append(rows, p.groupRows(p.CacheGroups, filter)...)
	}
	return rows
}

func (p *Project) sectionVisible(kind, filter string) bool {
	idx := p.ArtGroups
	if kind == "cache" {
		idx = p.CacheGroups
	}
	for _, gi := range idx {
		if p.groupLive(gi, filter) > 0 {
			return true
		}
	}
	return false
}

func (p *Project) groupLive(gi int, filter string) int {
	g := p.Groups[gi]
	n := 0
	for i := g.Start; i < g.Start+g.Len; i++ {
		it := p.Items[i]
		if it.Dead || !it.Matches(filter) {
			continue
		}
		n++
	}
	return n
}

func (p *Project) groupRows(idx []int, filter string) []ViewRow {
	var rows []ViewRow
	for _, gi := range idx {
		if p.groupLive(gi, filter) == 0 {
			continue
		}
		rows = append(rows, ViewRow{Kind: RowGroup, Ref: gi})
		g := p.Groups[gi]
		if g.Open || strings.TrimSpace(filter) != "" {
			for i := g.Start; i < g.Start+g.Len; i++ {
				it := p.Items[i]
				if it.Dead || !it.Matches(filter) {
					continue
				}
				rows = append(rows, ViewRow{Kind: RowItem, Ref: i})
			}
		}
	}
	return rows
}

func (p *Project) StateOf(pred func(Item) bool) (live, on int, bytes int64) {
	for _, it := range p.Items {
		if it.Dead || !pred(it) {
			continue
		}
		live++
		bytes += it.Size
		if it.On {
			on++
		}
	}
	return live, on, bytes
}

func BoxState(live, on int) string {
	if live > 0 && on == live {
		return "all"
	}
	if on > 0 {
		return "some"
	}
	return "none"
}

func (p *Project) ToggleMatching(pred func(Item) bool) {
	live, on, _ := p.StateOf(pred)
	next := true
	if live > 0 && on == live {
		next = false
	}
	for i := range p.Items {
		if p.Items[i].Dead || !pred(p.Items[i]) {
			continue
		}
		p.Items[i].On = next
	}
}

func (p *Project) ToggleItem(i int) {
	if i < 0 || i >= len(p.Items) || p.Items[i].Dead {
		return
	}
	p.Items[i].On = !p.Items[i].On
}

func (p *Project) SelectExpired(filter string) int {
	n := 0
	for i := range p.Items {
		it := &p.Items[i]
		if it.Kind != "artifact" || it.Dead || !it.Expired || !it.Matches(filter) {
			continue
		}
		it.On = true
		n++
	}
	return n
}

func (p *Project) SelectOlder(days int, filter string) int {
	if days < 0 {
		days = 0
	}
	cutoff := format.Now().Add(-time.Duration(days) * 24 * time.Hour)
	n := 0
	for i := range p.Items {
		it := &p.Items[i]
		if it.Kind != "artifact" || it.Dead || it.When.IsZero() || !it.Matches(filter) {
			continue
		}
		if it.When.After(cutoff) {
			continue
		}
		it.On = true
		n++
	}
	return n
}

// selectExceptNewest clears artifact checks, then selects every artifact
// except the newest keep copies of each name.
func (p *Project) SelectExceptNewest(keep int, filter string) int {
	for i := range p.Items {
		if p.Items[i].Kind == "artifact" {
			p.Items[i].On = false
		}
	}
	if keep < 0 {
		keep = 0
	}
	n := 0
	for _, gi := range p.ArtGroups {
		g := p.Groups[gi]
		rank := 0
		for i := g.Start; i < g.Start+g.Len; i++ {
			it := &p.Items[i]
			if it.Dead || !it.Matches(filter) {
				continue
			}
			if rank >= keep {
				it.On = true
				n++
			}
			rank++
		}
	}
	return n
}

func (p *Project) Selected() (n int, bytes int64, arts, caches int) {
	for _, it := range p.Items {
		if it.Dead || !it.On {
			continue
		}
		n++
		bytes += it.Size
		if it.Kind == "artifact" {
			arts++
		} else {
			caches++
		}
	}
	return n, bytes, arts, caches
}
