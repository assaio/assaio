package threshold

import (
	"strings"
	"testing"
	"time"
)

// fitting is a candidate whose every property matches -- the shape the register has none of
// today. The expiry rules have to be provable independently of whether anything currently fits,
// or the one mechanism that stops a citation rotting would be untested until the day it matters.
func fitting(validUntil time.Time) Candidate {
	fit := make([]Comparison, 0, len(Properties()))
	for _, p := range Properties() {
		fit = append(fit, Comparison{Property: p, Cited: "x", Assaio: "x", Same: true})
	}
	return Candidate{
		Metric: "example", Differs: "nothing", Fit: fit,
		Citation: Citation{
			Value: "1%", Definition: "d", Source: "s", URL: "https://example.test/paper",
			Population: "p",
			Measured:   date(2026, time.January, 1),
			Checked:    date(2026, time.January, 2),
			ValidUntil: validUntil,
		},
	}
}

// TestAppliesRequiresBothFitAndValidity is the register's central rule: a citation grades only
// while it is the same quantity *and* still in date, and either failing returns the verdict to
// withheld.
func TestAppliesRequiresBothFitAndValidity(t *testing.T) {
	expiry := date(2027, time.January, 1)
	unfit := fitting(expiry)
	unfit.Fit[2].Same = false

	for _, tc := range []struct {
		name      string
		candidate Candidate
		now       time.Time
		want      bool
	}{
		{"fits and in date", fitting(expiry), date(2026, time.June, 1), true},
		{"fits, one day before expiry", fitting(expiry), date(2026, time.December, 31), true},
		{"fits, expired", fitting(expiry), date(2027, time.January, 2), false},
		{"fits, expired by a second", fitting(expiry), expiry, false},
		{"fits, undated caller", fitting(expiry), time.Time{}, false},
		{"in date, does not fit", unfit, date(2026, time.June, 1), false},
		{"nothing compared", Candidate{}, date(2026, time.June, 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.candidate.Applies(tc.now); got != tc.want {
				t.Fatalf("Applies(%v) = %v, want %v", tc.now, got, tc.want)
			}
		})
	}
}

// TestDifferencesListsEveryUnmatchedPropertyInOrder checks the list a refusal renders: it is
// what tells a reader which part of the comparison broke.
func TestDifferencesListsEveryUnmatchedPropertyInOrder(t *testing.T) {
	c := fitting(date(2027, time.January, 1))
	c.Fit[4].Same = false
	c.Fit[0].Same = false
	got := c.Differences()
	want := []Property{Numerator, Layer}
	if len(got) != len(want) {
		t.Fatalf("Differences() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Differences() = %v, want %v", got, want)
		}
	}
}

// TestValidateRejectsAnIncompleteEntry is what makes a missing field a build failure rather
// than an empty string rendered beside a figure.
func TestValidateRejectsAnIncompleteEntry(t *testing.T) {
	for _, tc := range []struct {
		name string
		bend func(*Candidate)
		want string
	}{
		{"no metric", func(c *Candidate) { c.Metric = "" }, "leaves Metric empty"},
		{"no population difference", func(c *Candidate) { c.Differs = " " }, "leaves Differs empty"},
		{"no value", func(c *Candidate) { c.Citation.Value = "" }, "leaves Citation.Value empty"},
		{"no definition", func(c *Candidate) { c.Citation.Definition = "" }, "leaves Citation.Definition empty"},
		{"no source", func(c *Candidate) { c.Citation.Source = "" }, "leaves Citation.Source empty"},
		{"no population", func(c *Candidate) { c.Citation.Population = "" }, "leaves Citation.Population empty"},
		{"no url", func(c *Candidate) { c.Citation.URL = "" }, "leaves Citation.URL empty"},
		{"unresolvable url", func(c *Candidate) { c.Citation.URL = "gitclear.com" }, "cannot resolve"},
		{"no measurement date", func(c *Candidate) { c.Citation.Measured = time.Time{} }, "no measurement date"},
		{"never checked", func(c *Candidate) { c.Citation.Checked = time.Time{} }, "no date assaio last read"},
		{"checked before measured", func(c *Candidate) { c.Citation.Checked = date(2023, time.January, 1) }, "read before the measurement"},
		{"no expiry", func(c *Candidate) { c.Citation.ValidUntil = time.Time{} }, "expires no later than"},
		{"expires before it was measured", func(c *Candidate) { c.Citation.ValidUntil = date(2025, time.January, 1) }, "expires no later than"},
		{"a property never compared", func(c *Candidate) { c.Fit = c.Fit[:3] }, "never compares"},
		{"a property compared twice", func(c *Candidate) { c.Fit[1].Property = Numerator }, "compares numerator twice"},
		{"a comparison undescribed", func(c *Candidate) { c.Fit[1].Assaio = "" }, "leaves the denominator comparison undescribed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := fitting(date(2027, time.January, 1))
			tc.bend(&c)
			err := validate(&c)
			if err == nil {
				t.Fatalf("validate accepted an entry with %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validate error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
	if err := validate(&[]Candidate{fitting(date(2027, time.January, 1))}[0]); err != nil {
		t.Fatalf("validate rejected a complete entry: %v", err)
	}
}

// TestRegisteredCandidatesAreCheckableAtTheirSource holds the register's own rows to the rule
// the type enforces, and to the two that only a reader can check: an entry states which metric
// it was weighed against, and its Fit describes both sides of every property.
func TestRegisteredCandidatesAreCheckableAtTheirSource(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("the register is empty; a candidate that was adjudicated and then dropped leaves no record of the refusal")
	}
	for i := range all {
		c := &all[i]
		t.Run(c.Metric+"/"+c.Citation.Source, func(t *testing.T) {
			if err := validate(c); err != nil {
				t.Fatalf("registered entry is incomplete: %v", err)
			}
			if len(c.Fit) != len(Properties()) {
				t.Fatalf("Fit has %d comparisons, want one per defining property", len(c.Fit))
			}
		})
	}
}

// TestNoRegisteredCandidateGradesAnything records the state this register shipped in: every
// published figure assaio has weighed was refused. It is a fact about the evidence, not about
// the machinery, so it fails loudly the day one attaches -- at which point the entry has to be
// reviewed on its five properties rather than slipping in behind a green suite.
func TestNoRegisteredCandidateGradesAnything(t *testing.T) {
	now := date(2026, time.September, 2)
	all := All()
	for i := range all {
		c := &all[i]
		if c.Applies(now) {
			t.Fatalf("%s now grades against %s; review the five-property comparison and update this test deliberately",
				c.Metric, c.Citation.Source)
		}
	}
}

// TestCiteCarriesTheAddressWithoutItsScheme guards a contract that belongs to another package:
// the Assay dashboard renders these lines and fails its build on a literal "https://" anywhere
// in the file it produces. A citation that re-grew its scheme here would break a test nothing
// in this package names, so the rule is asserted where the string is built.
func TestCiteCarriesTheAddressWithoutItsScheme(t *testing.T) {
	all := All()
	for i := range all {
		c := &all[i]
		cite := c.Citation.Cite()
		if strings.Contains(cite, "https://") || strings.Contains(cite, "http://") {
			t.Fatalf("Cite() = %q, want no scheme: the offline dashboard must carry no external address", cite)
		}
		if !strings.Contains(cite, strings.TrimPrefix(c.Citation.URL, "https://")) {
			t.Fatalf("Cite() = %q, want it to name where the figure can be checked", cite)
		}
	}
}

// TestForSelectsByMetric checks the lookup a validator uses; a metric with nothing registered
// gets nothing rather than another metric's citation.
func TestForSelectsByMetric(t *testing.T) {
	if got := For("rework"); len(got) != 1 || got[0].Citation.Source != gitClearChurn.Source {
		t.Fatalf(`For("rework") = %+v, want the GitClear churn candidate`, got)
	}
	if got := For("burn"); len(got) != 0 {
		t.Fatalf(`For("burn") = %+v, want nothing registered`, got)
	}
}
