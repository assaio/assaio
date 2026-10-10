package attribution

const (
	renderDismissalUnavailable = "Dismissal coverage: — (unavailable; raw events may still be listed)."
	renderDismissalCoverage    = "Dismissal coverage: totalCount=%d (all timeline), filteredCount=%d (selected dismissals), pageCount=%d (source slice), listed=%d (retained nodes), hasPreviousPage=%t (selected truncation)."
	renderDismissalObservation = "Dismissal %s at %s: previous=%s review=%s commit=%s submitted=%s source=%s"
	renderDismissalCaveat      = "Raw source-local PR outcome from last 100 dismissals; no rounds or new rates. Current review snapshot unchanged."
)
