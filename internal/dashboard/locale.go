package dashboard

// localeStrings is the dashboard's static UI chrome: the eyebrows, headers, labels, and
// footer lines that read the same on every render, independent of the queried data.
// Figures, project/model names, and validator-generated Read/Takeaway/HowToRead text
// are data, not chrome, and never belong here -- they flow straight from analyze.Result.
//
// en is the only locale today. A future language switcher would add another
// localeStrings value (e.g. de, pl) and pick between them per request; there is no
// negotiation or switcher yet, by design -- see the locale template func in render.go,
// the one seam a switcher would hook into.
//
// Not every visible string routes through here. Three connector words are CSS
// `content:` values in dashboard.html.tmpl's <style> block (the "How to read — ", "Note
// — ", and "— " prefixes on .entry__howto/.entry__caveat/.entry__takeaway): they render
// from static stylesheet text, not from template data, so this seam does not reach them.
// The inline theme-toggle <script>'s own two aria-label strings are the same kind of
// exception, in JS rather than CSS. Both are deliberate, narrow carve-outs, not gaps.
type localeStrings struct {
	ReportEyebrow      string // masthead: "<ReportEyebrow> · <window>"
	StatusChip         string // masthead: local/offline status pill
	ToggleDarkLabel    string // masthead: theme toggle button's initial (light-mode) aria-label
	VerdictLabel       string // "Assay verdict" panel label and faceplate aria-label
	ProvLabel          string // faceplate cell / ledger entry: the "Prov." caveat-present stamp
	ReadsSuffix        string // panel label: "<n> <ReadsSuffix>"
	DimensionsSuffix   string // faceplate aria-label: "<n> <DimensionsSuffix>"
	AssayLabel         string // "Assay" -- all-projects panel label and drilldown title
	AllProjects        string // all-projects panel label
	ProjectsSuffix     string // panel label: "<n> <ProjectsSuffix>"
	SessionsLabel      string // panel label suffix and a stat/column label
	ActiveDaysSuffix   string // panel label: "<n> <ActiveDaysSuffix>"
	NoBarsInWindow     string // a ledger entry's Bars section, present but empty
	DrillSubtitle      string // drilldown section's one-line description
	AILinesLabel       string // a stat label and a subpath table column header
	SubpathBreakdown   string // subpath table eyebrow
	ScrollHint         string // subpath table's horizontal-scroll affordance hint
	SubpathColumn      string // subpath table's first column header
	NoSubpathData      string // subpath table empty state
	TotalRow           string // subpath table's footer row label
	CostBasisPrefix    string // footnote, before the computed cost-basis value
	CostBasisSuffix    string // footnote's fixed tagline, after the value
	UnpricedCaveat     string // colophon line, shown only when some cost was excluded as unpriced
	DirectionalCaveat  string // colophon line: the directional/privacy-posture note
	LineCoverageCaveat string // colophon line: which tools contribute AI-line counts
	QualityCaveat      string // colophon line: quality signals need the server stage
	AnonymizedCaveat   string // colophon line, shown only when project names are pseudonymized
	TeamPanelLabel     string // Team section's panel-label title, present only on a central store
	MembersSuffix      string // panel label: "<n> <MembersSuffix>"
	TeamCaption        string // Team section's one-line honesty caption
}

// en is today's -- and only -- locale.
var en = localeStrings{
	ReportEyebrow:      "Assay report",
	StatusChip:         "Local · Offline",
	ToggleDarkLabel:    "Switch to dark theme",
	VerdictLabel:       "Assay verdict",
	ProvLabel:          "Prov.",
	ReadsSuffix:        "reads",
	DimensionsSuffix:   "dimensions",
	AssayLabel:         "Assay",
	AllProjects:        "all projects",
	ProjectsSuffix:     "projects",
	SessionsLabel:      "sessions",
	ActiveDaysSuffix:   "active days",
	NoBarsInWindow:     "(none in this window)",
	DrillSubtitle:      "One project, broken down by repository subpath.",
	AILinesLabel:       "AI lines",
	SubpathBreakdown:   "Subpath breakdown",
	ScrollHint:         "· scrolls →",
	SubpathColumn:      "Subpath",
	NoSubpathData:      "No subpath breakdown available.",
	TotalRow:           "Total",
	CostBasisPrefix:    "Cost basis",
	CostBasisSuffix:    "the denominator, not the grade.",
	UnpricedCaveat:     "Cost figures marked * exclude usage on unpriced models -- a floor, not the full total.",
	DirectionalCaveat:  "Directional assay -- aggregate and pseudonymized by default; per-person requires an explicit opt-in (team mode).",
	LineCoverageCaveat: "AI-line signals: Claude Code and Codex today -- Gemini CLI and Cline contribute cost but not line counts (see ROADMAP).",
	QualityCaveat:      "Quality signals (survival in main, bug impact) need git/issue correlation -- server stage.",
	AnonymizedCaveat:   "Project names pseudonymized for sharing -- run with --no-anonymize for real names.",
	TeamPanelLabel:     "Team",
	MembersSuffix:      "members",
	TeamCaption:        "Team adoption -- aggregated, pseudonymized by default; per-person requires an explicit opt-in.",
}
