package analyze

import "sort"

// projectShare pairs a project's share of the window's tokens with its share of the
// window's AI lines. LineTokenShare is its share of the tokens from sources that record lines,
// the one the gap compares LineShare against; it is set only across named projects
// (namedProjects). Sorted by TokenShare descending.
type projectShare struct {
	Project        string
	TokenShare     float64
	LineTokenShare float64
	LineShare      float64
	Lines          int64
	// lineTokens and lineBlindTokens split the project's tokens by whether their source records
	// lines.
	lineTokens, lineBlindTokens int64
}

// projectShares converts the prepared per-project view into token-versus-line shares,
// dropping projects with no tokens: they contribute nothing to spend and would only
// dilute the concentration score with empty entries.
func projectShares(projects []ProjectStat, totalLines int64) []projectShare {
	out := make([]projectShare, 0, len(projects))
	for i := range projects {
		if projects[i].TokenShare <= 0 {
			continue
		}
		var lineShare float64
		if totalLines > 0 {
			lineShare = float64(projects[i].Lines) / float64(totalLines)
		}
		out = append(out, projectShare{
			Project:         projects[i].Project,
			TokenShare:      projects[i].TokenShare,
			LineShare:       lineShare,
			Lines:           projects[i].Lines,
			lineTokens:      projects[i].LineRate.OwnTokens(),
			lineBlindTokens: projects[i].LineRate.LineBlindTokens,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TokenShare > out[j].TokenShare })
	return out
}

// widestSpendGap returns the largest line-token-share-minus-line-share gap among projects big
// enough to matter (concentrationMinTokenShare). found is false when no project clears
// that floor, so a window of many tiny projects reports no gap rather than a loud one
// computed off noise. A project no source could have reported lines for has no line-recording
// tokens, so its share is zero and the floor skips it: its gap would measure the source.
func widestSpendGap(shares []projectShare) (gap float64, found bool) {
	for i := range shares {
		// The unattributed bucket is not a project: it is every tool that logs no working
		// directory, pooled. It writes no lines by construction, so scoring it would make
		// its whole token share the widest gap and blame a project that does not exist.
		if shares[i].Project == "" || shares[i].LineTokenShare < concentrationMinTokenShare {
			continue
		}
		if d := shares[i].LineTokenShare - shares[i].LineShare; !found || d > gap {
			gap, found = d, true
		}
	}
	return gap, found
}

// giniOfShares scores dispersion across projects 0..1 -- 0 when every project holds an
// equal share, approaching 1 when one holds nearly all of it. Uses the sorted-rank form of
// the Gini coefficient, normalized by the shares' own sum so it stays correct when they do
// not total exactly 1.
func giniOfShares(shares []projectShare) float64 {
	n := len(shares)
	if n < concentrationMinProjects {
		return 0
	}
	ascending := make([]float64, n)
	for i := range shares {
		ascending[i] = shares[i].TokenShare
	}
	sort.Float64s(ascending)
	var sum, weighted float64
	for i, s := range ascending {
		sum += s
		weighted += float64(i+1) * s
	}
	if sum == 0 {
		return 0
	}
	return clamp01(2*weighted/(float64(n)*sum) - float64(n+1)/float64(n))
}

// topShare is the largest project's share of the window's tokens.
func topShare(shares []projectShare) float64 { return topNShare(shares, 1) }

// topNShare sums the token share of the n largest projects; shares is already sorted
// descending.
func topNShare(shares []projectShare, n int) float64 {
	var sum float64
	for i := range shares {
		if i >= n {
			break
		}
		sum += shares[i].TokenShare
	}
	return sum
}

// namedProjects drops the unattributed bucket and renormalizes what is left, so every
// concentration statistic reads as a share of attributable spend and output. Without the
// rescale the headline shares would stay fractions of the whole window while giniOfShares
// divides by the survivors' own sum -- two figures on the same line, computed against
// different denominators, disagreeing about how concentrated the same window is.
func namedProjects(shares []projectShare) []projectShare {
	out := make([]projectShare, 0, len(shares))
	var tokens, lines float64
	var lineTokens int64
	for i := range shares {
		if shares[i].Project == "" {
			continue
		}
		out = append(out, shares[i])
		tokens += shares[i].TokenShare
		lines += shares[i].LineShare
		lineTokens += shares[i].lineTokens
	}
	for i := range out {
		out[i].TokenShare = rescale(out[i].TokenShare, tokens)
		out[i].LineShare = rescale(out[i].LineShare, lines)
		out[i].LineTokenShare = rescale(float64(out[i].lineTokens), float64(lineTokens))
	}
	return out
}

// rescale expresses share as a fraction of total, 0 when nothing was attributed.
func rescale(share, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return share / total
}

// unattributedShare is the token share of usage with no resolved project name -- the
// Gemini CLI and Cline case, whose logs carry no working directory.
func unattributedShare(shares []projectShare) float64 {
	for i := range shares {
		if shares[i].Project == "" {
			return shares[i].TokenShare
		}
	}
	return 0
}
