package analyze

import "github.com/assaio/assaio/internal/layer"

// The label kinds Result.BarsPseudonym may name. Each is a dimension whose values people
// choose themselves -- a repository, a skill, a sub-agent -- and can therefore identify the
// work an anonymized report is meant to hide.
const (
	PseudonymProject = "project"
	PseudonymSkill   = "skill"
)

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
// report and HTML dashboard render from. Every field is plain data with no behavior,
// so a server-side view can populate the same shape from aggregated data and reuse the
// same renderer.
type Result struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Describe string `json:"describe"`
	Read     Read   `json:"read"`
	// Layer is which of the four measurement layers this verdict rests on, stamped by Evaluate
	// from the Validator's own declaration so a Result cannot claim a layer its metric did not.
	// A figure inside the Result may sit on another layer as context; the label states what the
	// verdict is a claim about, which is what a reader acts on.
	Layer layer.Layer `json:"layer"`
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
	// BarsPseudonym names the kind of user-authored label Bars carry -- PseudonymProject or
	// PseudonymSkill -- and is what makes the dashboard pseudonymize them under --anonymize
	// (see internal/dashboard.anonymizeVerdicts). Leave "" (the default) when Bars label a
	// fixed vocabulary the tool itself defines (models, tools, time bands, ...); those must
	// never be pseudonymized. Set it from any Validator, built-in or custom, whose Bars rank
	// by a name a person chose -- the dashboard applies the rule generically.
	BarsPseudonym string `json:"barsPseudonym,omitempty"`
	Takeaway      string `json:"takeaway"`
	// Withheld is the declared inputs (see Needs) this window could not supply. Empty for a
	// validator that declared nothing, which is most of them while the migration is in
	// progress. It is the difference a reader acts on between a metric that measured its
	// subject and found nothing and one that was never handed the evidence to look at.
	Withheld []Capability `json:"withheld,omitempty"`
	// Caveats are honesty notes (directional, contested, or server-stage-only signals);
	// optional.
	Caveats []string `json:"caveats,omitempty"`
	// Confidence is what this verdict rests on: coverage, how many observations it counted,
	// and how current the data is. A validator sets only Confidence.Samples and Unit; the
	// rest is stamped by Evaluate, which is how every Result carries it -- including a
	// validator written out of tree.
	Confidence Confidence `json:"confidence"`
	// Lead marks the few reads worth a week's attention, set by MarkLead and absent
	// everywhere else -- including on every read of a window that promoted none, which is a
	// real answer rather than a missing one. A validator never sets it: which findings lead
	// is a question about the window, and no metric can see the others.
	Lead *Lead `json:"lead,omitempty"`
}

// Lead is a read's place in the window's ordering and the reasons that put it there.
type Lead struct {
	Rank    int      `json:"rank"`
	Reasons []string `json:"reasons"`
}
