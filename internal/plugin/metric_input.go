package plugin

import (
	"time"

	"github.com/assaio/assaio/internal/analyze"
)

// metricInputVersion is the stdin envelope's protocol version, carried in the
// assaio_metric_input key.
// v2 is the first change a plugin cannot absorb silently: a result must declare its `layer`
// (ADR 0013). Every field added before it was additive, so the version stayed at 1 through
// v0.17 and v0.21; a newly required output field has to be a version a plugin can branch on,
// or its only signal is a contract violation naming a field it has never heard of.
//
// v3 is the second: `trace` is now sent only to a plugin whose config declares `needs: [trace]`
// (B168). A plugin built against v2 read the timeline unconditionally, so leaving the version
// alone would have had it report "no sequences" over a full store -- a wrong number with no
// error attached, which is the failure mode this project refuses. Bumping makes the handshake
// fail loudly instead, naming the version, so the fix is a config line rather than a
// mystery.
//
// v4 is the third, and the first to reshape the *request* rather than the answer: a plugin
// declares what it reads in a `describe` run, and the envelope carries only that -- the sections
// it named, the columns inside them, and the rows its own predicates admit. A v3 plugin declared
// nothing and was sent everything, so leaving the version alone would hand it a document with
// most of its sections missing and no way to tell a projection from an empty window. Bumping
// fails the handshake instead, naming the version, so the fix is a `describe` verb rather than a
// figure computed over sections that were never there.
const metricInputVersion = 4

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
	// By far the largest section -- 339,000 steps encode to about 44 MB -- which is what a
	// projection is for: a detector that reads three step columns of one scope no longer pays
	// for the other four columns and the other three scopes. Every scope is sent unless a
	// predicate excludes one, each sequence carrying the scope it belongs to, because a
	// detector's scope is its denominator and both sides have to agree on it rather than each
	// deriving one.
	//
	// Empty on a store with no step history, and for every source with no step reading: a
	// window that recorded nothing, never a session that did nothing.
	Trace []metricTimeline `json:"trace"`
	// Withheld names the capabilities the plugin declared and the local config denied. It is
	// the only absence assaio decided: a section missing because the plugin never asked for it
	// is named by Projection.Needs instead. Both exist because an empty array is
	// indistinguishable from a window that holds none, and a plugin dividing by one would be
	// reporting a zero nobody measured.
	Withheld []analyze.Capability `json:"withheld,omitempty"`
	// Projection is what this run carries and why: the capabilities granted, the columns and
	// predicates the plugin declared, and per section how many rows it received out of how many
	// the window held. It is what makes the document self-describing, which a projected one has
	// to be -- a column projected away is absent, and nothing else on the wire says whether an
	// absence was chosen or measured.
	Projection metricProjection `json:"projection"`
	// HistoryStart is the earliest observation the store holds, ignoring this window. It is what
	// makes a trend's own horizon knowable: a comparison against an earlier span means nothing when
	// the store's history began inside it, and a source that deletes its transcripts makes that the
	// ordinary case. The zero time means the core could not answer.
	HistoryStart time.Time `json:"historyStart"`
}

// metricProjection echoes the negotiated projection back to the plugin that declared it, after
// the local config's veto.
type metricProjection struct {
	Needs  []analyze.Capability `json:"needs"`
	Fields map[string][]string  `json:"fields,omitempty"`
	Where  map[string][]string  `json:"where,omitempty"`
	// Rows counts, per array section this envelope carries, how many rows the plugin received
	// out of how many the window held before its own predicate ran. A share computed over a
	// filtered section has its denominator here and nowhere else.
	Rows map[string]rowCount `json:"rows"`
}
