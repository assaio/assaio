package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// concentrationInput seeds two projects whose token and line shares diverge by the given
// amounts: "hungry" burns most of the tokens, "lean" writes most of the lines.
func concentrationInput(hungryTokens, hungryLines, leanTokens, leanLines int64) Input {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "hungry", In: hungryTokens, LinesAdded: hungryLines},
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "lean", In: leanTokens, LinesAdded: leanLines},
	}
	return BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
}

// TestConcentrationFlagsSpendNotConvertingToOutput covers the metric's whole point: a
// project taking 80% of the tokens while writing 10% of the lines is the gap worth
// flagging, not the concentration itself.
func TestConcentrationFlagsSpendNotConvertingToOutput(t *testing.T) {
	got := mustGet(t, concentrationName).Analyze(concentrationInput(8000, 10, 2000, 90))

	if got.Read.Label != "WATCH" {
		t.Fatalf("Read = %q, want WATCH for an 80%%-tokens / 10%%-lines project", got.Read.Label)
	}
	if !strings.Contains(figureValues(got.Figures), "70pp") {
		t.Fatalf("Figures = %q, want the 70pp spend gap", figureValues(got.Figures))
	}
	if !strings.Contains(got.Takeaway, "larger share of tokens") {
		t.Fatalf("Takeaway = %q, want the spend-versus-output message", got.Takeaway)
	}
}

// TestConcentrationAlignedWhenSpendTracksOutput asserts that heavy concentration alone is
// not a fault: a project on most of the tokens that also wrote most of the lines reads as
// aligned.
func TestConcentrationAlignedWhenSpendTracksOutput(t *testing.T) {
	got := mustGet(t, concentrationName).Analyze(concentrationInput(8000, 80, 2000, 20))

	if got.Read.Label != "ALIGNED" {
		t.Fatalf("Read = %q, want ALIGNED when token share matches line share", got.Read.Label)
	}
}

// TestConcentrationSingleProjectIsUndefined asserts a lone project yields the neutral
// no-verdict read rather than a favorable one earned by having nothing to compare against.
func TestConcentrationSingleProjectIsUndefined(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "solo", In: 5000, LinesAdded: 100},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, concentrationName).Analyze(in)

	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the neutral no-verdict read for a single project", got.Read)
	}
	if !strings.Contains(got.Takeaway, "at least two") {
		t.Fatalf("Takeaway = %q, want the single-project explanation", got.Takeaway)
	}
}

// TestConcentrationFiguresNeverNameAProject locks the privacy boundary: only Bars are
// pseudonymized under --anonymize, so a project name reaching a Figure or the Takeaway
// would leak the very name an anonymized dashboard promises to hide.
func TestConcentrationFiguresNeverNameAProject(t *testing.T) {
	got := mustGet(t, concentrationName).Analyze(concentrationInput(8000, 10, 2000, 90))

	for _, f := range got.Figures {
		for _, field := range []string{f.Label, f.Value, f.Note} {
			for _, project := range []string{"hungry", "lean"} {
				if strings.Contains(field, project) {
					t.Fatalf("Figure field %q names project %q; only Bars are anonymized", field, project)
				}
			}
		}
	}
	if strings.Contains(got.Takeaway, "hungry") || strings.Contains(got.Takeaway, "lean") {
		t.Fatalf("Takeaway = %q must not name a project", got.Takeaway)
	}
	if got.BarsPseudonym != PseudonymProject {
		t.Fatal("BarsPseudonym must be set so the dashboard pseudonymizes these bars")
	}
}

// TestConcentrationCaveatsUnattributedTokens asserts usage that resolved to no project --
// the Gemini/Cline case, which log no working directory -- is disclosed rather than
// silently ranked as a project named "(unknown)".
func TestConcentrationCaveatsUnattributedTokens(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "gemini-cli", Model: "claude-sonnet-4-5", Project: "", In: 5000},
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 5000, LinesAdded: 100},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, concentrationName).Analyze(in)

	if len(got.Caveats) == 0 || !strings.Contains(strings.Join(got.Caveats, " "), "unattributed") {
		t.Fatalf("Caveats = %q, want the unattributed-tokens disclosure", got.Caveats)
	}
}

// TestGiniOfShares checks the dispersion score at both ends: an even split scores ~0 and a
// near-monopoly approaches 1.
func TestGiniOfShares(t *testing.T) {
	cases := []struct {
		name   string
		shares []float64
		wantLo float64
		wantHi float64
	}{
		{"even split of four", []float64{0.25, 0.25, 0.25, 0.25}, 0, 0.01},
		{"near monopoly", []float64{0.97, 0.01, 0.01, 0.01}, 0.7, 1},
		{"single project", []float64{1}, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			shares := make([]projectShare, len(tc.shares))
			for i, s := range tc.shares {
				shares[i] = projectShare{TokenShare: s}
			}
			got := giniOfShares(shares)
			if got < tc.wantLo || got > tc.wantHi {
				t.Fatalf("giniOfShares = %v, want within [%v, %v]", got, tc.wantLo, tc.wantHi)
			}
		})
	}
}

// TestConcentrationEmptyInputSafe asserts the zero-value Input renders the honest no-data
// block rather than a verdict computed from nothing.
func TestConcentrationEmptyInputSafe(t *testing.T) {
	got := mustGet(t, concentrationName).Analyze(Input{})
	if got.Read != noDataRead || got.Takeaway != "No usage in this window." {
		t.Fatalf("empty Input = %+v, want the no-data read and takeaway", got)
	}
}
