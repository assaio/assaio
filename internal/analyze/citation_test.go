package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/threshold"
)

func citationDate(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// citationFixture is a candidate whose every property matches. The register has none today, so
// the applied-line rendering and the expiry that removes it can only be proven against a
// fixture -- and they have to be, since the expiry is what keeps a citation from grading
// forever after its source has moved on.
func citationFixture(same bool, validUntil time.Time) threshold.Candidate {
	fit := make([]threshold.Comparison, 0, len(threshold.Properties()))
	for _, p := range threshold.Properties() {
		fit = append(fit, threshold.Comparison{Property: p, Cited: "c", Assaio: "a", Same: same})
	}
	return threshold.Candidate{
		Metric: "example", Differs: "a different population", Fit: fit,
		Citation: threshold.Citation{
			Value: "9%", Definition: "what the study counts", Source: "A Study",
			// No full stop: the clause is finished by the formatter, and a fixture that ended
			// itself would let a formatter that does not still pass.
			URL: "https://example.test/paper", Population: "some repositories",
			Measured:   citationDate(2026, time.January, 1),
			Checked:    citationDate(2026, time.January, 2),
			ValidUntil: validUntil,
		},
	}
}

// TestCitationLinesGradeOnlyWhileFitAndInDate is the expiry demonstration: the same candidate
// that carries a line before its validity ends withholds after it, with the reader told which.
func TestCitationLinesGradeOnlyWhileFitAndInDate(t *testing.T) {
	expiry := citationDate(2027, time.January, 1)
	for _, tc := range []struct {
		name      string
		candidate threshold.Candidate
		now       time.Time
		want      string
		absent    string
	}{
		{
			name: "fits and in date", candidate: citationFixture(true, expiry),
			now:  citationDate(2026, time.June, 1),
			want: "Line: A Study (example.test/paper), measured through 2026-01-01: 9%",
		},
		{
			name: "the same citation, expired", candidate: citationFixture(true, expiry),
			now:  citationDate(2027, time.June, 1),
			want: "stopped counting on 2027-01-01", absent: "Line: A Study",
		},
		{
			name: "an undated caller is never handed a live citation", candidate: citationFixture(true, expiry),
			now:  time.Time{},
			want: "stopped counting on 2027-01-01", absent: "Line: A Study",
		},
		{
			name: "in date but a different quantity", candidate: citationFixture(false, expiry),
			now:  citationDate(2026, time.June, 1),
			want: "numerator, denominator, window, data source, layer differ", absent: "Line: A Study",
		},
		// The case every registered candidate is in, and the one a fit-first switch could never
		// reach: an unfit pairing whose citation has since expired must stop being named too,
		// or the refusal keeps quoting a figure the source has restated.
		{
			name: "unfit and expired stops being named", candidate: citationFixture(false, expiry),
			now:  citationDate(2027, time.June, 1),
			want: "stopped counting on 2027-01-01",
			// The property list is the unfit rendering; past expiry the study is not held up
			// against this figure at all, so it must not appear.
			absent: "numerator, denominator, window, data source, layer differ",
		},
		{
			name: "the population clause is finished by the formatter", candidate: citationFixture(false, expiry),
			now:  citationDate(2026, time.June, 1),
			want: "over some repositories. It counts",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(citationLines([]threshold.Candidate{tc.candidate}, tc.now), "\n")
			if !strings.Contains(got, tc.want) {
				t.Fatalf("citationLines = %q, want it to contain %q", got, tc.want)
			}
			if tc.absent != "" && strings.Contains(got, tc.absent) {
				t.Fatalf("citationLines = %q, want it not to contain %q", got, tc.absent)
			}
		})
	}
	if got := citationLines(nil, citationDate(2026, time.June, 1)); got != nil {
		t.Fatalf("citationLines(nil) = %v, want nothing said about a comparison nobody registered", got)
	}
}

// TestPopulationEndsItsOwnClause: the full stop between the population and the sentence after it
// belongs to the formatter. A register entry that supplies one must not render two, and one that
// supplies none must not run the two sentences together.
func TestPopulationEndsItsOwnClause(t *testing.T) {
	for _, tc := range []struct{ name, population, want string }{
		{"no stop of its own", "some repositories", "over some repositories. "},
		{"its own stop", "some repositories.", "over some repositories. "},
		{"a trailing space", "some repositories. ", "over some repositories. "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := citationFixture(true, citationDate(2027, time.January, 1))
			c.Citation.Population = tc.population
			if got := population(&c); got != tc.want {
				t.Fatalf("population = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestReworkNamesTheChurnStudyItIsNotBeingGradedAgainst is why the register renders at all:
// assaio prints a rework percentage and GitClear prints a churn percentage, and a reader told
// nothing will subtract one from the other.
func TestReworkNamesTheChurnStudyItIsNotBeingGradedAgainst(t *testing.T) {
	got := reworkFixtureResult(t)
	joined := strings.Join(got.Caveats, "\n")
	for _, want := range []string{
		"Not a line here",
		"GitClear",
		"www.gitclear.com/ai_assistant_code_quality_2025_research",
		"reverted or substantially revised within the subsequent two weeks",
		"numerator, denominator, window, data source, layer differ",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("rework caveats = %q, want them to contain %q", joined, want)
		}
	}
	if strings.Contains(joined, "Line: GitClear") {
		t.Fatalf("rework caveats = %q, want no line drawn from a study that measures something else", joined)
	}
	if got.Read != reportedRead {
		t.Fatalf("Read = %+v, want the verdict still withheld", got.Read)
	}
}

// reworkFixtureResult runs the rework validator over a window that records both halves of the
// pair, which is the window a reader would be comparing against a published churn rate.
func reworkFixtureResult(t *testing.T) Result {
	t.Helper()
	usage := []store.UsageRow{{
		Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
		In: 10, Out: 10, LinesAdded: 100, ReworkLines: 10, ToolCalls: 20, Rejected: 1,
	}}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	return reworkValidator{}.Analyze(in)
}
