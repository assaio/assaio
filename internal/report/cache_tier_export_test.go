package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"math"
	"strconv"
	"testing"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

const tierModel = "claude-opus-4-5"

var tierPrice = pricing.Price{Input: 5e-6, Output: 2.5e-5, CacheWrite: 6.25e-6, CacheRead: 5e-7, CacheWrite1h: 1e-5}

// tierUsage mixes a source that states the cache tier (claude-code) with one that records cache
// writes and no tier (cline), on one model so a grouped row folds both.
func tierUsage() []store.UsageRow {
	return []store.UsageRow{
		{Day: "2026-07-01", Tool: "claude-code", Model: tierModel, In: 100, Out: 200, CacheRead: 800, CacheWrite: 500, CacheWrite1h: 300},
		{Day: "2026-07-02", Tool: "claude-code", Model: tierModel, In: 10, Out: 20, CacheRead: 80, CacheWrite: 50, CacheWrite1h: 50},
		{Day: "2026-07-02", Tool: "cline", Model: tierModel, In: 7, Out: 9, CacheRead: 30, CacheWrite: 40},
	}
}

// TestExportedColumnsReproduceTheCost: a machine export whose own token columns cannot explain
// its cost is a figure nobody can check. The 1-hour cache write is billed at its own rate, so
// both formats carry it, and an aggregate keeps summing it.
func TestExportedColumnsReproduceTheCost(t *testing.T) {
	tbl := pricing.Table{tierModel: tierPrice}
	grouped, err := Aggregate(Build(tierUsage(), tbl), "model")
	if err != nil {
		t.Fatal(err)
	}
	for name, rows := range map[string][]Row{"by day": Build(tierUsage(), tbl), "by model": grouped} {
		t.Run(name+"/csv", func(t *testing.T) {
			for _, got := range csvExport(t, rows) {
				assertReproduced(t, got)
			}
		})
		t.Run(name+"/json", func(t *testing.T) {
			for _, got := range jsonExport(t, rows) {
				assertReproduced(t, got)
			}
		})
	}
}

// TestUntieredWritesAreNotReportedAsShortLived: a source that records cache writes and no tier
// has not said its writes were the 5-minute kind, so its 1-hour figure is absent rather than 0,
// and a grouped row's share can only be taken over the writes that stated a tier.
func TestUntieredWritesAreNotReportedAsShortLived(t *testing.T) {
	tbl := pricing.Table{tierModel: tierPrice}
	for _, e := range jsonExport(t, Build(tierUsage(), tbl)) {
		if e.Tool == "cline" && (e.CacheWrite1h != nil || e.CacheWriteTiered != 0) {
			t.Fatalf("untiered source exported a tier: %+v", e)
		}
	}
	var buf bytes.Buffer
	if err := RenderCSV(&buf, Build(tierUsage(), tbl)); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range records[1:] {
		if rec[columnIndex(t, records[0], "tool")] != "cline" {
			continue
		}
		long, tiered := rec[columnIndex(t, records[0], "cache_write_1h")], rec[columnIndex(t, records[0], "cache_write_tiered")]
		if long != "" || tiered != "0" {
			t.Fatalf("untiered CSV row: cache_write_1h %q, cache_write_tiered %q; want an empty cell and 0", long, tiered)
		}
	}
	grouped, err := Aggregate(Build(tierUsage(), tbl), "model")
	if err != nil {
		t.Fatal(err)
	}
	g := jsonExport(t, grouped)[0]
	if g.CacheWrite != 590 || g.CacheWriteTiered != 550 || g.CacheWrite1h == nil || *g.CacheWrite1h != 350 {
		t.Fatalf("grouped row = %+v, want cache_write 590, tiered 550, 1h 350", g)
	}
}

type exported struct {
	Tool             string  `json:"tool"`
	In               int64   `json:"in"`
	Out              int64   `json:"out"`
	CacheRead        int64   `json:"cache_read"`
	CacheWrite       int64   `json:"cache_write"`
	CacheWrite1h     *int64  `json:"cache_write_1h"`
	CacheWriteTiered int64   `json:"cache_write_tiered"`
	Cost             float64 `json:"cost"`
}

func jsonExport(t *testing.T, rows []Row) []exported {
	t.Helper()
	var buf bytes.Buffer
	if err := RenderJSON(&buf, rows); err != nil {
		t.Fatal(err)
	}
	var decoded []exported
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func csvExport(t *testing.T, rows []Row) []exported {
	t.Helper()
	var buf bytes.Buffer
	if err := RenderCSV(&buf, rows); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	header := records[0]
	if n := len(header); header[n-2] != "cache_write_1h" || header[n-1] != "cache_write_tiered" {
		t.Fatalf("the tier columns must come last, so positional readers keep working: %v", header)
	}
	num := func(rec []string, name string) int64 {
		v, err := strconv.ParseInt(rec[columnIndex(t, header, name)], 10, 64)
		if err != nil {
			t.Fatalf("column %s: %v", name, err)
		}
		return v
	}
	var out []exported
	for _, rec := range records[1:] {
		cost, err := strconv.ParseFloat(rec[columnIndex(t, header, "cost")], 64)
		if err != nil {
			t.Fatal(err)
		}
		e := exported{
			Tool: rec[columnIndex(t, header, "tool")],
			In:   num(rec, "in"), Out: num(rec, "out"), CacheRead: num(rec, "cache_read"),
			CacheWrite: num(rec, "cache_write"), CacheWriteTiered: num(rec, "cache_write_tiered"), Cost: cost,
		}
		if rec[columnIndex(t, header, "cache_write_1h")] != "" {
			long := num(rec, "cache_write_1h")
			e.CacheWrite1h = &long
		}
		out = append(out, e)
	}
	return out
}

func assertReproduced(t *testing.T, e exported) {
	t.Helper()
	var long int64
	if e.CacheWrite1h != nil {
		long = *e.CacheWrite1h
	}
	p := tierPrice
	want := float64(e.In)*p.Input + float64(e.Out)*p.Output + float64(e.CacheRead)*p.CacheRead +
		float64(e.CacheWrite-long)*p.CacheWrite + float64(long)*p.CacheWrite1h
	if math.Abs(want-e.Cost) > 1e-6 {
		t.Fatalf("cost from exported columns = %.6f, exported cost = %.6f (%+v)", want, e.Cost, e)
	}
}
