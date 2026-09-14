package analyze

import (
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/trace"
)

// Delegation is the real sub-agent token-delegation share: Sub is tokens on records
// whose dedupe_key marks a Task sub-agent turn, Total is every token in the same window
// (see internal/store.Store.Delegation). The CLI populates this so validators stay pure
// functions of Input rather than reaching into the store themselves.
type Delegation struct {
	Sub, Total int64
}

// Input is the read-only bundle every Validator draws from; each Validator uses only the
// fields its metric needs. ByModel, ByProject, and Totals are pre-computed, ready-to-use
// aggregates -- built once by BuildInput -- and are what most validators, built-in or
// custom, should read: reaching into raw Usage to re-group by model/project, or importing
// internal/report to do it, is rarely needed once these are populated. Usage and Sessions
// remain for anything the prepared views don't cover, e.g. day-level or session-grain
// signals (see model_fit.go for a validator built entirely on ByModel, and throughput.go,
// context.go, or rework.go for ones that still need Usage/Sessions directly). Recent is
// the recent-vs-prior window (e.g. 7d) validators use for trend and staleness signals.
type Input struct {
	// WindowStart is the --since boundary Usage was queried with. It is what a monthly
	// projection divides by: a window is a span of real days, and the days inside it that
	// happen to carry no usage are still days a flat plan was paid for. Zero means the caller
	// scoped no window, and a projection then spans the usage itself.
	WindowStart time.Time
	Usage       []store.UsageRow
	Sessions    []store.SessionRow
	Prices      pricing.Table
	Now         time.Time
	Recent      time.Duration
	Delegation  Delegation
	// ByModel is Usage aggregated per model, tier-classified and priced, sorted by
	// Tokens descending. See ModelStat.
	ByModel []ModelStat
	// ByProject is Usage aggregated per project, sorted by Lines descending. See
	// ProjectStat.
	ByProject []ProjectStat
	// Totals is Usage's grand totals across every model and project. See Totals.
	Totals Totals
	// PlanMonthlyCost is the user's flat monthly subscription price (config.pricing), for
	// comparing against the API-equivalent estimate. Zero (the default) means unset, and
	// subscription-fit then prompts to configure it rather than comparing against nothing.
	PlanMonthlyCost float64
	// Skills and Agents are the window's per-skill and per-sub-agent totals from
	// store.Attribution, each sorted by Tokens descending. Populated by the CLI; empty in
	// the drill and in tests that don't set them, where skill-economics reports that no
	// usage carried attribution rather than inventing a zero.
	Skills []store.AttributionRow
	Agents []store.AttributionRow
	// TurnSizing is per-model raw turn counts (output-producing turns, and how many were
	// small) for model-right-sizing, which needs the per-turn grain the daily Usage
	// aggregate hides. Populated by the CLI from store.TurnSizing; empty in the drill and
	// in tests that don't set it, where model-right-sizing reads "no premium turns".
	TurnSizing []store.ModelTurns
	// CacheMisses is the window's stated cache-miss reasons per tool, from
	// store.CacheMisses, ordered by turns descending. Populated by the CLI; empty in the
	// drill and in tests that don't set it, where cache-hygiene reports no stated cause
	// rather than inventing one.
	CacheMisses []store.CacheMissRow
	// Trace is the window's stored step sequences (internal/trace): what each session did, in
	// what order. A validator reading it asks for the one scope it declares rather than the
	// whole set, so its rate never spans an interactive session and a one-shot SDK call. Empty
	// in tests that don't set it and on every source with no step reading, where a detector reports
	// that no sequence was stored rather than inventing a zero. The drill narrows it to its own
	// project's sessions rather than leaving it empty (internal/dashboard.buildDrill), so a
	// project panel's detectors answer for that project.
	Trace trace.Set
	// HistoryStart is the earliest observation the store holds, ignoring this window's --since. It
	// is what makes a trend's own horizon knowable: a comparison against an earlier span is
	// meaningless when the store's history began inside it, which after a source deletes its
	// transcripts is the ordinary case rather than the odd one (B156, Trending). Zero means the
	// caller could not answer.
	HistoryStart time.Time
	// Ingested is when the newest data in the store was read and ParsedBy is the build that
	// read it; both travel onto every Result's Confidence. Zero and "" mean unknown, which
	// is what a caller that cannot answer should leave them as rather than guessing.
	Ingested time.Time
	ParsedBy string
}
