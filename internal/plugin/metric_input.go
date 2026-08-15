package plugin

import "time"

// metricInputVersion is the stdin envelope's protocol version, carried in the
// assaio_metric_input key.
// v2 is the first change a plugin cannot absorb silently: a result must declare its `layer`
// (ADR 0013). Every field added before it was additive, so the version stayed at 1 through
// v0.17 and v0.21; a newly required output field has to be a version a plugin can branch on,
// or its only signal is a contract violation naming a field it has never heard of.
const metricInputVersion = 2

// MetricProtocolVersion is metricInputVersion for the published-docs check, so a recipe and the
// runtime cannot disagree about which handshake a plugin must emit.
const MetricProtocolVersion = metricInputVersion

// metricInput is the wire envelope a metric plugin reads on stdin: analyze.Input mapped
// onto explicit wire types so a core refactor never silently changes the protocol -- the
// same decoupling wireRecord gives the parser protocol (ADR 0003/0004). Field names are
// camelCase to mirror the public `analyze --format json` Result shape; only the version
// keys stay snake_case, matching assaio_plugin.
type metricInput struct {
	Version    int                    `json:"assaio_metric_input"`
	Now        time.Time              `json:"now"`
	RecentDays int                    `json:"recentDays"`
	Usage      []metricUsageRow       `json:"usage"`
	Sessions   []metricSessionRow     `json:"sessions"`
	Delegation metricDelegation       `json:"delegation"`
	ByModel    []metricModelStat      `json:"byModel"`
	ByProject  []metricProjectStat    `json:"byProject"`
	Totals     metricTotals           `json:"totals"`
	Prices     map[string]metricPrice `json:"prices"`
	// Answers maps every tool present in this window to the signal ids it can produce
	// (internal/signal, `assaio-agent signals list`). Every count on a row below is zero
	// for a source that does not record it, and "nothing happened" and "nothing was
	// written down" are different facts: a metric over one of those columns must keep only
	// the rows whose tool answers the matching signal, exactly as a built-in validator does
	// (ADR 0011). Without this the wire made that impossible to get right.
	Answers map[string][]string `json:"answers"`
	// WindowStart is the --since boundary usage was queried with; the zero time means the
	// caller scoped no window, and a rate then spans the usage itself rather than the
	// window. A projection over real days divides by this, because a day inside the window
	// carrying no usage is still a day a flat plan was paid for.
	WindowStart time.Time `json:"windowStart"`
	// PlanMonthlyCost is the configured flat subscription price. Zero means nobody
	// configured one, never a plan that costs nothing.
	PlanMonthlyCost float64 `json:"planMonthlyCost"`
	// Skills and Agents are the window's per-skill and per-sub-agent totals. Rows carrying
	// no attribution are absent rather than bucketed under an empty name, and both are
	// empty when no tool in the window records attribution at all -- which is a coverage
	// fact to state, not a zero to publish.
	Skills []metricAttributionRow `json:"skills"`
	Agents []metricAttributionRow `json:"agents"`
	// TurnSizing is per-model turn counts at the raw per-turn grain the daily usage rows
	// aggregate away, so a metric about turns measures turns rather than day totals.
	TurnSizing []metricModelTurns `json:"turnSizing"`
	// CacheMisses is the window's stated cache-miss reasons per tool. A turn that hit cache
	// states no reason and is absent, as is every turn from a source that reports none.
	CacheMisses []metricCacheMissRow `json:"cacheMisses"`
	// Trace is the window's step sequences: what each session did, in what order (ADR 0012).
	// Every scope is sent, each sequence carrying the one it belongs to, because a detector's
	// scope is its denominator and both sides have to agree on it rather than each deriving one.
	//
	// This is the largest thing on the wire by an order of magnitude -- 339,000 steps encode to
	// 44MB -- and it is sent unconditionally because the alternative is a plugin that cannot
	// write the detectors the core just gained. A plugin declaring what it needs is the way out
	// and needs a protocol version to carry it (`B168`).
	//
	// Empty on a store with no step history, and for every source with no step reading: absent,
	// not a session that did nothing.
	Trace []metricTimeline `json:"trace"`
	// HistoryStart is the earliest observation the store holds, ignoring this window. It is what
	// makes a trend's own horizon knowable: a comparison against an earlier span means nothing when
	// the store's history began inside it, and a source that deletes its transcripts makes that the
	// ordinary case. The zero time means the core could not answer.
	HistoryStart time.Time `json:"historyStart"`
}

type metricUsageRow struct {
	Day        string `json:"day"`
	Tool       string `json:"tool"`
	Model      string `json:"model"`
	Project    string `json:"project"`
	Entrypoint string `json:"entrypoint"`
	Member     string `json:"member"`
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
	SessionID         string    `json:"sessionId"`
	Project           string    `json:"project"`
	Tool              string    `json:"tool"`
	Model             string    `json:"model"`
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
