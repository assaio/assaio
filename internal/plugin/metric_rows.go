package plugin

// The row and aggregate shapes the envelope's sections are made of, kept apart from the
// envelope itself so the protocol's documented surface reads as one file and its columns as
// another. Every field here is addressable by a projection (metric_section.go), which is why
// each one carries a tagged, exported, non-embedded column name.

import "time"

type metricUsageRow struct {
	Day        string `json:"day"`
	Tool       string `json:"tool"`
	Model      string `json:"model"`
	Project    string `json:"project"`
	Entrypoint string `json:"entrypoint"`
	// Member is a pseudonym, never the name a member synced under -- see memberLabels.
	Member string `json:"member"`
	// Granularity is "turn" or "session". A plugin that sums these rows without reading it
	// would fold whole-session aggregates into a per-turn figure, which is the same misread
	// the core reports now disclose.
	Granularity string `json:"granularity"`
	In          int64  `json:"in"`
	Out         int64  `json:"out"`
	CacheRead   int64  `json:"cacheRead"`
	CacheWrite  int64  `json:"cacheWrite"`
	// CacheWrite1h is the portion of CacheWrite that bought a 1-hour cache lifetime, billed
	// at its own higher rate (metricPrice.CacheWrite1h). A subset, never added to the total.
	// Without it a plugin re-pricing these rows necessarily bills every write at the cheaper
	// 5-minute rate and reports a cost the core does not agree with.
	CacheWrite1h int64 `json:"cacheWrite1h"`
	Reasoning    int64 `json:"reasoning"`
	LinesAdded   int64 `json:"linesAdded"`
	LinesRemoved int64 `json:"linesRemoved"`
	Edits        int64 `json:"edits"`
	ToolCalls    int64 `json:"toolCalls"`
	Rejected     int64 `json:"rejected"`
	Compactions  int64 `json:"compactions"`
	ReworkLines  int64 `json:"reworkLines"`
}

type metricSessionRow struct {
	SessionID string `json:"sessionId"`
	Project   string `json:"project"`
	Tool      string `json:"tool"`
	Model     string `json:"model"`
	// Member is a pseudonym, never the name a member synced under -- see memberLabels.
	Member            string    `json:"member"`
	FirstTs           time.Time `json:"firstTs"`
	LastTs            time.Time `json:"lastTs"`
	Turns             int64     `json:"turns"`
	OutputTokens      int64     `json:"outputTokens"`
	PeakContextTokens int64     `json:"peakContextTokens"`
	Edits             int64     `json:"edits"`
	Compactions       int64     `json:"compactions"`
	ActiveMinutes     float64   `json:"activeMinutes"`
}

type metricDelegation struct {
	Sub   int64 `json:"sub"`
	Total int64 `json:"total"`
}

type metricModelStat struct {
	Model      string   `json:"model"`
	Tier       string   `json:"tier"`
	Tokens     int64    `json:"tokens"`
	Input      int64    `json:"input"`
	Output     int64    `json:"output"`
	CacheRead  int64    `json:"cacheRead"`
	CacheWrite int64    `json:"cacheWrite"`
	Lines      int64    `json:"lines"`
	Cost       *float64 `json:"cost"`
	Priced     bool     `json:"priced"`
	TokenShare float64  `json:"tokenShare"`
}

type metricProjectStat struct {
	Project    string   `json:"project"`
	Lines      int64    `json:"lines"`
	Cost       *float64 `json:"cost"`
	Priced     bool     `json:"priced"`
	TokenShare float64  `json:"tokenShare"`
}

type metricTotals struct {
	Tokens          int64    `json:"tokens"`
	Input           int64    `json:"input"`
	Output          int64    `json:"output"`
	CacheRead       int64    `json:"cacheRead"`
	CacheWrite      int64    `json:"cacheWrite"`
	Lines           int64    `json:"lines"`
	Cost            *float64 `json:"cost"`
	Priced          bool     `json:"priced"`
	CacheEfficiency float64  `json:"cacheEfficiency"`
}

type metricPrice struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	// CacheWrite1h prices the 1-hour portion of a cache write; it equals CacheWrite for a
	// model the table gives only one write rate, which is the vendor charging one rate rather
	// than the tier being free.
	CacheWrite1h float64 `json:"cacheWrite1h"`
}
