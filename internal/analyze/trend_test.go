package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// trendTestNow puts the recent span on Jul 7–13 and the earlier one on Jun 30–Jul 6.
var trendTestNow = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

func trendRow(day, tool string, lines, tokens int64) store.UsageRow {
	return store.UsageRow{Day: day, Tool: tool, Model: "claude-sonnet-4-5", LinesAdded: lines, In: tokens}
}

// running is a row before both weeks: the source was already in use when the earlier one began,
// so its week there is whole.
func running(tool string) store.UsageRow { return trendRow("2026-06-20", tool, 0, 0) }

func trendRows() []store.UsageRow {
	return []store.UsageRow{
		trendRow("2026-07-10", "claude-code", 200, 0),
		trendRow("2026-07-08", "claude-code", 50, 0),
		trendRow("2026-07-01", "claude-code", 100, 0),
		running("claude-code"),
	}
}

// trendInput is a 30-day window over a store reaching back 60 days, so neither the window nor
// the horizon stops a direction unless a case says so.
func trendInput(rows []store.UsageRow) Input {
	return Input{
		Usage: rows, Now: trendTestNow, Recent: 7 * 24 * time.Hour,
		WindowStart: trendTestNow.AddDate(0, 0, -30), HistoryStart: trendTestNow.AddDate(0, 0, -60),
	}
}

func TestTrendSpansAreCompleteDaysEndingYesterday(t *testing.T) {
	recent, prior := trendSpans(trendTestNow, 7*24*time.Hour)
	if recent != (span{"2026-07-07", "2026-07-13"}) || prior != (span{"2026-06-30", "2026-07-06"}) {
		t.Fatalf("spans = %+v / %+v, want Jul 7–13 and Jun 30–Jul 6", recent, prior)
	}
	if got := prior.label() + " → " + recent.label(); got != "Jun 30–Jul 6 → Jul 7–13" {
		t.Fatalf("labels = %q", got)
	}
	recent, prior = trendSpans(time.Date(2027, 1, 5, 3, 0, 0, 0, time.UTC), 7*24*time.Hour)
	if got := prior.label() + " → " + recent.label(); got != "Dec 22–28 → Dec 29–Jan 4" {
		t.Fatalf("labels across the year = %q", got)
	}
}

// The floor is the window's own typical day: a week that burned less than one ordinary day
// prints no direction, and the same rows a week busier print one.
func TestBurnTrendWithholdsUnderOneTypicalDay(t *testing.T) {
	rows := func(recent int64) []store.UsageRow {
		out := []store.UsageRow{trendRow("2026-07-01", "claude-code", 0, 300), trendRow("2026-07-10", "claude-code", 0, recent)}
		for d := 1; d <= 20; d++ {
			out = append(out, trendRow(time.Date(2026, 6, d, 0, 0, 0, 0, time.UTC).Format(time.DateOnly), "claude-code", 0, 10_000))
		}
		return out
	}
	get := func(recent int64) Figure {
		in := BuildInput(rows(recent), nil, testPrices(), trendTestNow, 7*24*time.Hour, Delegation{})
		return findFigure(t, mustGet(t, burnName).Analyze(in).Figures, tokensTrend.label)
	}
	if f := get(400); f.Value != "—" || !strings.Contains(f.Note, "300 → 400") || !strings.Contains(f.Note, "too small") {
		t.Fatalf("figure = %+v, want a dash with both sums and the floor named", f)
	}
	if f := get(20_000); f.Value != "+6567%" {
		t.Fatalf("figure = %+v, want a direction once the busier week clears one typical day", f)
	}
}

func TestReadTrend(t *testing.T) {
	with := func(extra ...store.UsageRow) []store.UsageRow { return append(trendRows(), extra...) }
	cases := []struct {
		name      string
		rows      []store.UsageRow
		edit      func(*Input)
		wantValue string
		wantNote  string
	}{
		{"both weeks read", trendRows(), nil, "+150%", "Jun 30–Jul 6 → Jul 7–13 · 100 → 250"},
		{"today belongs to neither week", with(trendRow("2026-07-14", "claude-code", 900, 0)), nil, "+150%", "100 → 250"},
		{
			"the earlier week's first day counts, the day before does not",
			with(trendRow("2026-06-30", "claude-code", 20, 0), trendRow("2026-06-29", "claude-code", 5000, 0)), nil, "+108%", "120 → 250",
		},
		{
			"a window opening inside the earlier week", trendRows(),
			func(in *Input) { in.WindowStart = trendTestNow.AddDate(0, 0, -10) }, "—", "window starts after Jun 30",
		},
		{
			"a window opening at the earlier week's first midnight holds it", trendRows(),
			func(in *Input) { in.WindowStart = time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC) }, "+150%", "100 → 250",
		},
		{
			"history that begins inside the earlier week", trendRows(),
			func(in *Input) { in.HistoryStart = time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC) }, "—", "store history begins Jul 2",
		},
		{
			"an unreadable history still reads, with its caveat elsewhere", trendRows(),
			func(in *Input) { in.HistoryStart = time.Time{} }, "+150%", "100 → 250",
		},
		{
			"an earlier week with no rows at all",
			[]store.UsageRow{trendRow("2026-07-10", "claude-code", 200, 0), trendRow("2026-07-08", "claude-code", 50, 0), running("claude-code")},
			nil,
			"—", "0 → 250 · earlier-week volume is zero; no percentage exists",
		},
		{
			"no source ran before the earlier week", trendRows()[:3], nil,
			"—", "Jun 30–Jul 6 → Jul 7–13 · no source has a row by Jun 30 in this window",
		},
		{
			"an earlier week with nothing written in it",
			[]store.UsageRow{trendRow("2026-07-10", "claude-code", 250, 0), trendRow("2026-07-01", "claude-code", 0, 0), running("claude-code")},
			nil,
			"—", "0 → 250 · earlier-week volume is zero; no percentage exists",
		},
		{
			"both weeks too small",
			[]store.UsageRow{trendRow("2026-07-10", "claude-code", 2, 0), trendRow("2026-07-01", "claude-code", 1, 0), running("claude-code")},
			nil,
			"—", "1 → 2 · volume is too small to state a direction",
		},
		{
			"a collapse to nothing stays readable",
			[]store.UsageRow{trendRow("2026-07-10", "claude-code", 0, 0), trendRow("2026-07-01", "claude-code", 1000, 0), running("claude-code")},
			nil,
			"-100%", "1,000 → 0",
		},
		{
			"a source first seen inside the comparison is left out of both", with(trendRow("2026-07-09", "codex", 5000, 0)), nil,
			"+150%", "100 → 250 · 1 source: no row by Jun 30 in window; omitted (93% of both weeks' lines)",
		},
		{
			"a source that stopped still counts",
			[]store.UsageRow{running("claude-code"), running("codex"), trendRow("2026-07-01", "claude-code", 1000, 0), trendRow("2026-07-02", "codex", 400, 0)},
			nil,
			"-100%", "1,400 → 0",
		},
		{
			"a source that records no lines is not counted at all", with(trendRow("2026-07-09", "gemini-cli", 0, 900)), nil,
			"+150%", "Jun 30–Jul 6 → Jul 7–13 · 100 → 250",
		},
		{
			"a source whose first row falls on the earlier week's first day counts",
			[]store.UsageRow{trendRow("2026-06-30", "claude-code", 100, 0), trendRow("2026-07-10", "claude-code", 250, 0)},
			nil,
			"+150%", "100 → 250",
		},
		{
			"a source whose first row falls the day after is left out, with its share",
			with(trendRow("2026-07-01", "codex", 400, 0), trendRow("2026-07-09", "codex", 600, 0)), nil,
			"+150%", "100 → 250 · 1 source: no row by Jun 30 in window; omitted (74% of both weeks' lines)",
		},
		{
			"a window opening a second into the earlier week's first day withholds, and carries no sums", trendRows(),
			func(in *Input) { in.WindowStart = time.Date(2026, 6, 30, 0, 0, 1, 0, time.UTC) },
			"—", "Jun 30–Jul 6 → Jul 7–13 · window starts after Jun 30",
		},
		{
			"history that begins on the earlier week's first day, at any hour, covers it", trendRows(),
			func(in *Input) { in.HistoryStart = time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC) }, "+150%", "100 → 250",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := trendInput(c.rows)
			if c.edit != nil {
				c.edit(&in)
			}
			tr := readTrend(&in, linesTrend, linesTrendFloor)
			f := tr.figure(linesTrend)
			if f.Label != "week-over-week AI lines" || f.Value != c.wantValue || !strings.Contains(f.Note, c.wantNote) {
				t.Fatalf("figure = %+v, want value %q with note containing %q", f, c.wantValue, c.wantNote)
			}
			if c.wantNote == "Jun 30–Jul 6 → Jul 7–13 · 100 → 250" && f.Note != c.wantNote {
				t.Fatalf("note = %q, want exactly %q", f.Note, c.wantNote)
			}
		})
	}
}

func TestChangeLabelNeverRoundsAChangeToZero(t *testing.T) {
	cases := []struct {
		now, was int64
		want     string
	}{
		{250, 100, "+150%"},
		{100, 100, "0%"},
		{1_000_000, 1_004_000, "-<1%"},
		{1005, 1000, "+<1%"},
		{1006, 1000, "+1%"},
		{0, 100, "-100%"},
		{5, 0, "—"},
	}
	for _, c := range cases {
		if got := changeLabel(c.now, c.was); got != c.want {
			t.Errorf("changeLabel(%d, %d) = %q, want %q", c.now, c.was, got, c.want)
		}
	}
}

// The figures below are properties a correct trend keeps whatever its inputs: scale, a shift of
// the calendar, and a source that cannot answer the question do not move a direction.
func TestTokensTrendKeepsItsDirectionUnderScale(t *testing.T) {
	rows := []store.UsageRow{trendRow("2026-07-10", "claude-code", 0, 3000), trendRow("2026-07-01", "claude-code", 0, 2000), running("claude-code")}
	in := trendInput(rows)
	base := readTrend(&in, tokensTrend, 0)
	for i := range rows {
		rows[i].In *= 1000
	}
	in = trendInput(rows)
	scaled := readTrend(&in, tokensTrend, 0)
	if base.figure(tokensTrend).Value != "+50%" || scaled.figure(tokensTrend).Value != "+50%" {
		t.Fatalf("values = %q / %q, want +50%% both", base.figure(tokensTrend).Value, scaled.figure(tokensTrend).Value)
	}
}

func TestTrendMovesWithTheCalendar(t *testing.T) {
	shifted := trendRows()
	for i := range shifted {
		d, _ := time.Parse(time.DateOnly, shifted[i].Day)
		shifted[i].Day = d.AddDate(0, 0, 1).Format(time.DateOnly)
	}
	in := trendInput(shifted)
	in.Now = in.Now.AddDate(0, 0, 1)
	in.WindowStart, in.HistoryStart = in.WindowStart.AddDate(0, 0, 1), in.HistoryStart.AddDate(0, 0, 1)
	tr := readTrend(&in, linesTrend, linesTrendFloor)
	if f := tr.figure(linesTrend); f.Value != "+150%" || f.Note != "Jul 1–7 → Jul 8–14 · 100 → 250" {
		t.Fatalf("figure = %+v, want the same direction one day later", f)
	}
}

func TestTokensTrendIgnoresASourceWithNoTokenCounter(t *testing.T) {
	rows := []store.UsageRow{trendRow("2026-07-10", "claude-code", 0, 3000), trendRow("2026-07-01", "claude-code", 0, 2000), running("claude-code")}
	in := trendInput(rows)
	base := readTrend(&in, tokensTrend, 0)
	want := base.figure(tokensTrend)
	in = trendInput(append(rows, trendRow("2026-07-11", "agy", 0, 0), trendRow("2026-07-02", "agy", 0, 0)))
	withAgy := readTrend(&in, tokensTrend, 0)
	if got := withAgy.figure(tokensTrend); got != want {
		t.Fatalf("figure = %+v, want %+v unchanged by a source that counts no tokens", got, want)
	}
}

// TestAdoptionAndThroughputShareTrendComputation: given the same Usage/Now/Recent, adoption's and
// throughput's week-over-week figures must agree exactly (same helper, same numbers).
func TestAdoptionAndThroughputShareTrendComputation(t *testing.T) {
	in := BuildInput(trendRows(), nil, testPrices(), trendTestNow, 7*24*time.Hour, Delegation{})
	adoptionResult := mustGet(t, "adoption").Analyze(in)
	throughputResult := mustGet(t, "throughput").Analyze(in)

	adoptionTrend := findFigure(t, adoptionResult.Figures, linesTrend.label)
	throughputTrend := findFigure(t, throughputResult.Figures, linesTrend.label)
	if adoptionTrend != throughputTrend {
		t.Fatalf("adoption trend figure %+v != throughput trend figure %+v", adoptionTrend, throughputTrend)
	}
}

// Swapping the two weeks leaves the multiset of daily burns, its median and its spikes where
// they were and turns the trend around. Read and Purity must not notice.
func TestBurnReadAndPurityIgnoreTheTrend(t *testing.T) {
	quiet := []int64{900, 1000, 1100, 1000, 950, 1050, 1000}
	busy := []int64{2000, 2100, 1900, 2000, 2050, 1950, 2000}
	burn := func(earlier, recent []int64) Result {
		var usage []store.UsageRow
		for i, tokens := range append(append([]int64{}, earlier...), recent...) {
			day := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i).Format(time.DateOnly)
			usage = append(usage, store.UsageRow{Day: day, Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: tokens})
		}
		return mustGet(t, burnName).Analyze(BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{}))
	}
	a, b := burn(quiet, busy), burn(busy, quiet)
	if a.Read != b.Read || a.Purity != b.Purity {
		t.Fatalf("read/purity moved with the trend: %+v %v vs %+v %v", a.Read, a.Purity, b.Read, b.Purity)
	}
	if findFigure(t, a.Figures, tokensTrend.label).Value == findFigure(t, b.Figures, tokensTrend.label).Value {
		t.Fatalf("the trend did not turn around: %+v", findFigure(t, a.Figures, tokensTrend.label))
	}
}

func TestTrendPurityNeutralWhenUnknown(t *testing.T) {
	if got := trendPurity(0, false); got != 0.5 {
		t.Fatalf("trendPurity(_, false) = %v, want 0.5 (neutral)", got)
	}
}

func TestTrendPuritySaturatesAtExtremes(t *testing.T) {
	if got := trendPurity(5, true); got != 1 {
		t.Fatalf("trendPurity(+500%%) = %v, want 1 (clamped)", got)
	}
	if got := trendPurity(-5, true); got != 0 {
		t.Fatalf("trendPurity(-500%%) = %v, want 0 (clamped)", got)
	}
}

func mustGet(t *testing.T, name string) Validator {
	t.Helper()
	v, ok := Get(name)
	if !ok {
		t.Fatalf("validator %q not registered", name)
	}
	return v
}

func findFigure(t *testing.T, figures []Figure, label string) Figure {
	t.Helper()
	for _, f := range figures {
		if f.Label == label {
			return f
		}
	}
	t.Fatalf("no figure labeled %q in %+v", label, figures)
	return Figure{}
}
