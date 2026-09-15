package attribution

// Summarize counts outcomes and names how much of the population had any candidate at all.
func Summarize(results []Result) Summary {
	summary := Summary{Population: len(results)}
	for i := range results {
		switch results[i].Status {
		case statusMatched:
			summary.Matched++
		case statusAmbiguous:
			summary.Ambiguous++
		default:
			summary.Unmatched++
		}
	}
	summary.CandidateCovered = summary.Matched + summary.Ambiguous
	if summary.Population > 0 {
		summary.CandidateCoverage = float64(summary.CandidateCovered) / float64(summary.Population)
		summary.ResolvedCoverage = float64(summary.Matched) / float64(summary.Population)
	}
	return summary
}
