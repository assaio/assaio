package attribution

import "testing"

func TestSummarizeUsesOneExplicitPopulation(t *testing.T) {
	results := []Result{
		{Status: statusMatched},
		{Status: statusAmbiguous},
		{Status: statusUnmatched},
		{Status: statusUnmatched},
	}
	got := Summarize(results)
	if got.Population != 4 || got.Matched != 1 || got.Ambiguous != 1 || got.Unmatched != 2 {
		t.Fatalf("counts = %#v", got)
	}
	if got.CandidateCovered != 2 || got.CandidateCoverage != 0.5 || got.ResolvedCoverage != 0.25 {
		t.Fatalf("coverage = %#v", got)
	}
}
