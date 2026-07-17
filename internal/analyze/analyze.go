// Package analyze is assaio's validator framework. Each Validator reads the same Input
// bundle and returns one structured Result -- the single source of truth the CLI text
// report and a future HTML dashboard both render from. Adding a metric is a one-file
// change: implement Validator and call Register from that file's init() -- see
// "Adding a metric validator" in docs/extending.md.
package analyze

import (
	"sort"
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
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
	Usage      []store.UsageRow
	Sessions   []store.SessionRow
	Prices     pricing.Table
	Now        time.Time
	Recent     time.Duration
	Delegation Delegation
	// ByModel is Usage aggregated per model, tier-classified and priced, sorted by
	// Tokens descending. See ModelStat.
	ByModel []ModelStat
	// ByProject is Usage aggregated per project, sorted by Lines descending. See
	// ProjectStat.
	ByProject []ProjectStat
	// Totals is Usage's grand totals across every model and project. See Totals.
	Totals Totals
}

// Read is a validator's headline verdict. Key drives the dashboard's color: "good",
// "watch", or "neutral" for a window with no data. Label is the short word the text
// report shows in brackets, already upper-cased, e.g. "STRONG", "WATCH".
type Read struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Figure is one stat: a big Value with a Label caption, e.g. {"projects", "32", ""}.
// Note is an optional short parenthetical -- a secondary number or a "*" marker.
type Figure struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Note  string `json:"note,omitempty"`
}

// Bar is one row of a ranked-list visualization (top projects, model split, ...): a
// Label, its display Value, and Frac (0..1) scaled against that list's own maximum.
type Bar struct {
	Label string  `json:"label"`
	Value string  `json:"value"`
	Frac  float64 `json:"frac"`
}

// Result is a Validator's complete output: the single structured shape the CLI text
// report and a future HTML dashboard both render from. Every field is plain data with no
// behavior, so a later server-side view can populate the same shape from aggregated data
// and reuse the same renderer.
type Result struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Describe string `json:"describe"`
	Read     Read   `json:"read"`
	// Purity is 0..1 for the dashboard's faceplate gauge: how "well-used" this
	// dimension reads, set honestly per validator (see each validator's *Purity func).
	Purity float64 `json:"purity"`
	// HowToRead is a one-sentence, plain-language explainer of what this dimension
	// means and what to do with it -- always populated, even on a "no data" Result, so
	// the CLI's "? " line and the dashboard's ledger helper line always have context to
	// show. The single source both surfaces render from; see RenderResultText and
	// dashboard.html.tmpl's ledgerEntry.
	HowToRead string   `json:"howToRead"`
	Figures   []Figure `json:"figures,omitempty"`
	Bars      []Bar    `json:"bars,omitempty"`
	// BarsAreProjects marks whether Bars' Label values are project names -- the one
	// dimension the dashboard pseudonymizes under --anonymize (see
	// internal/dashboard.anonymizeVerdicts). Leave false (the default) when Bars label
	// anything else (models, tools, ...); those must never be pseudonymized. Set it from
	// any Validator, built-in or custom, whose Bars rank by project -- the dashboard
	// applies the rule generically, not just to the built-in throughput validator.
	BarsAreProjects bool   `json:"barsAreProjects,omitempty"`
	Takeaway        string `json:"takeaway"`
	// Caveats are honesty notes (directional, contested, or server-stage-only signals);
	// optional.
	Caveats []string `json:"caveats,omitempty"`
}

// Validator is one independently testable, self-describing metric. Name is a stable
// kebab-case slug (e.g. "model-fit") used on the command line and as the JSON key;
// Title is a human label; Describe is a one-line summary for `assaio analyze --list`.
type Validator interface {
	Name() string
	Title() string
	Describe() string
	Analyze(Input) Result
}

// registry holds every self-registered Validator. Populated by each validator file's
// init(); never mutated after program start.
var registry []Validator

// Register adds v to the set of validators `assaio analyze` runs. This is the extension
// point for a new metric: add a file under internal/analyze implementing Validator and
// call Register(yourValidator{}) from that file's init(). No other wiring is required --
// the validator appears in `assaio analyze --list` and every analyze run automatically.
func Register(v Validator) {
	registry = append(registry, v)
}

// Validators returns every registered Validator, sorted by Name for stable,
// deterministic output across runs.
func Validators() []Validator {
	out := make([]Validator, len(registry))
	copy(out, registry)
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Get returns the validator registered under name, if any.
func Get(name string) (Validator, bool) {
	for _, v := range registry {
		if v.Name() == name {
			return v, true
		}
	}
	return nil, false
}
