package drift

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// run builds one recorded ingest pass; fields left at zero are the uninteresting ones for
// the canary under test.
func run(discovered, parsed, records, skipped, zeroToken int) store.SourceRun {
	return store.SourceRun{
		Tool: "claude-code", RanAt: time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC),
		Discovered: discovered, Parsed: parsed, Records: records,
		Skipped: skipped, ZeroToken: zeroToken,
	}
}

// repeat builds a baseline of n identical runs, the history a current run is judged against.
func repeat(n int, r *store.SourceRun) []store.SourceRun {
	out := make([]store.SourceRun, 0, n)
	for range n {
		out = append(out, *r)
	}
	return out
}

// fired reports whether ws contains the named canary.
func fired(ws []Warning, canary string) bool {
	for _, w := range ws {
		if w.Canary == canary {
			return true
		}
	}
	return false
}

func TestCanariesFireOnTheirSignal(t *testing.T) {
	healthy := run(5742, 40, 840, 0, 1)
	tests := []struct {
		name    string
		history []store.SourceRun
		current store.SourceRun
		canary  string
	}{
		{
			name:    "discovery: the source vanished entirely",
			history: repeat(4, &healthy),
			current: run(0, 0, 0, 0, 0),
			canary:  Discovery,
		},
		{
			name:    "discovery: less than half the files are still found",
			history: repeat(4, &healthy),
			current: run(2000, 40, 840, 0, 1),
			canary:  Discovery,
		},
		{
			name:    "yield: files were read but produced nothing",
			history: repeat(4, &healthy),
			current: run(5742, 40, 0, 0, 0),
			canary:  Yield,
		},
		{
			name:    "yield: records per file collapsed against history",
			history: repeat(4, &healthy),
			current: run(5742, 40, 80, 0, 0),
			canary:  Yield,
		},
		{
			name:    "skipped: lines stopped unmarshaling",
			history: repeat(4, &healthy),
			current: run(5742, 40, 1400, 400, 0),
			canary:  Skipped,
		},
		{
			name:    "zero-token: records parse but carry no tokens",
			history: repeat(4, &healthy),
			current: run(5742, 40, 2000, 0, 1900),
			canary:  ZeroToken,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(append(tt.history, tt.current))
			if !fired(got, tt.canary) {
				t.Fatalf("canary %q did not fire; got %+v", tt.canary, got)
			}
		})
	}
}

// TestCanariesAbstainOnThinData is the honesty rule applied to the canaries themselves: a
// ratio computed from a handful of files or records is not evidence, so below the sample
// floor they say nothing rather than guess.
func TestCanariesAbstainOnThinData(t *testing.T) {
	small, healthy := run(10, 4, 80, 0, 0), run(5742, 40, 840, 0, 1)
	tests := []struct {
		name    string
		history []store.SourceRun
		current store.SourceRun
	}{
		{
			name:    "discovery: baseline too small to read a partial drop",
			history: repeat(4, &small),
			current: run(3, 2, 40, 0, 0),
		},
		{
			name:    "yield: too few files parsed to judge records per file",
			history: repeat(4, &healthy),
			current: run(5742, 5, 10, 0, 0),
		},
		{
			name:    "skipped: a handful of skipped lines is not a trend",
			history: repeat(4, &healthy),
			current: run(5742, 2, 6, 4, 0),
		},
		{
			name:    "zero-token: too few records to read a share",
			history: repeat(4, &healthy),
			current: run(5742, 2, 10, 0, 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Evaluate(append(tt.history, tt.current)); len(got) != 0 {
				t.Fatalf("want silence on thin data, got %+v", got)
			}
		})
	}
}

// TestAbsoluteCanariesFireOnTheVeryFirstRun keeps a broken parser detectable before any
// baseline exists -- the skipped and zero-token shares are read against what a healthy
// parse looks like, not against this install's history.
func TestAbsoluteCanariesFireOnTheVeryFirstRun(t *testing.T) {
	got := Evaluate([]store.SourceRun{run(5742, 5742, 2000, 0, 1900)})
	if !fired(got, ZeroToken) {
		t.Fatalf("zero-token canary needs no history; got %+v", got)
	}
}

// TestDiscoveryCanaryGoesQuietOnceTheNewNormalSettles stops a genuinely uninstalled tool
// from warning forever: once the retained history is all zeroes, zero is the baseline.
func TestDiscoveryCanaryGoesQuietOnceTheNewNormalSettles(t *testing.T) {
	quiet := run(0, 0, 0, 0, 0)
	history := repeat(6, &quiet)
	if got := Evaluate(append(history, run(0, 0, 0, 0, 0))); len(got) != 0 {
		t.Fatalf("want silence once zero is the norm, got %+v", got)
	}
}

func TestWarningCarriesToolCanaryAndNumbers(t *testing.T) {
	healthy := run(5742, 40, 840, 0, 1)
	got := Evaluate(append(repeat(4, &healthy), run(0, 0, 0, 0, 0)))
	if len(got) == 0 {
		t.Fatal("want a warning")
	}
	w := got[0]
	if w.Tool != "claude-code" {
		t.Errorf("Tool = %q, want claude-code", w.Tool)
	}
	if w.Canary != Discovery {
		t.Errorf("Canary = %q, want %q", w.Canary, Discovery)
	}
	if !strings.Contains(w.Detail, "5742") {
		t.Errorf("Detail must show the baseline it was judged against, got %q", w.Detail)
	}
}

func TestEvaluateEmptyHistory(t *testing.T) {
	if got := Evaluate(nil); got != nil {
		t.Fatalf("want nil for no runs, got %+v", got)
	}
}

// TestSkippedCanaryDoesNotDivideOneUnitByAnother holds the unit rule: Records counts emitted records, one
// per API response and several lines each, while Skipped counts unreadable lines, undated
// records and refused steps. Their sum is not a line count, so a 3% real skip rate reported
// itself as 10.7% and cleared the threshold on arithmetic alone.
func TestSkippedCanaryDoesNotDivideOneUnitByAnother(t *testing.T) {
	healthy := run(5742, 40, 840, 0, 1)
	// 200 files yielding 500 records and 60 unreadable lines: 0.3 skips per file, and the old
	// share of Records+Skipped read 10.7%.
	current := run(200, 200, 500, 60, 0)
	if got := Evaluate(append(repeat(4, &healthy), current)); fired(got, Skipped) {
		t.Fatalf("the skipped canary fired on a rate it computed from two different units: %+v", got)
	}
}

// TestBarrenCanaryFiresOnlyOnASourceThatNeverYielded: a source that has always yielded zero has
// a baseline of zero, so every history-comparing canary abstains and no sample floor changes
// that -- verified by A/B on the real corpus with all four floors set to 1, where both builds
// reported "no canary fired". The condition is the whole history rather than one run, because
// Parsed counts files a run attempted: an incremental pass whose one changed input yields
// nothing must not report a healthy source as barren.
func TestBarrenCanaryFiresOnlyOnASourceThatNeverYielded(t *testing.T) {
	healthy := run(6300, 40, 840, 0, 1)
	cases := []struct {
		name    string
		history []store.SourceRun
		current store.SourceRun
		want    bool
	}{
		{
			name:    "the live gemini-cli case: files found, nothing ever read out of them",
			history: nil,
			current: run(2, 2, 0, 0, 0),
			want:    true,
		},
		{
			name:    "a history of always yielding zero still fires",
			history: repeat(4, &store.SourceRun{Tool: "gemini-cli", Discovered: 2, Parsed: 2}),
			current: run(2, 2, 0, 0, 0),
			want:    true,
		},
		{
			name:    "an incremental pass whose one changed file yields nothing is not barren",
			history: repeat(4, &healthy),
			current: run(6300, 1, 0, 0, 0),
			want:    false,
		},
		{
			name:    "one record out is not barren",
			history: nil,
			current: run(2, 2, 1, 0, 0),
			want:    false,
		},
		{
			name:    "nothing discovered is the discovery canary's case, not this one",
			history: nil,
			current: run(0, 0, 0, 0, 0),
			want:    false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(append(tt.history, tt.current))
			if fired(got, Barren) != tt.want {
				t.Fatalf("barren fired = %v, want %v; got %+v", fired(got, Barren), tt.want, got)
			}
		})
	}
}
