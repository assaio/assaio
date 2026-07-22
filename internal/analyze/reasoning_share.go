package analyze

const (
	reasoningName      = "reasoning-share"
	reasoningTitle     = "Reasoning Share"
	reasoningDescribe  = "How much generated output is extended-thinking (reasoning) tokens, among tools that report it -- flagging deep reasoning spent on shallow tasks."
	reasoningHowToRead = "Reasoning tokens are billed as output. A high share means much of the model's work is internal deliberation, which can be overkill on routine tasks. Only some tools report reasoning (Codex and Gemini CLI today), so the coverage figure says how much of your output this even covers."
	// reasoningWatchShare is the reasoning-of-output share above which the read flags heavy thinking.
	reasoningWatchShare = 0.3
)

func init() { Register(reasoningValidator{}) }

// reasoningValidator reports the reasoning share of output among tools that surface it, and
// how much of the window's output that covers, so a Claude-heavy window reads honestly.
type reasoningValidator struct{}

func (reasoningValidator) Name() string     { return reasoningName }
func (reasoningValidator) Title() string    { return reasoningTitle }
func (reasoningValidator) Describe() string { return reasoningDescribe }

//nolint:gocritic // Input is required by the Validator interface; analyzed once per run, not a hot path.
func (reasoningValidator) Analyze(in Input) Result {
	r := Result{Name: reasoningName, Title: reasoningTitle, Describe: reasoningDescribe, HowToRead: reasoningHowToRead}
	if in.Totals.Output == 0 {
		r.Read = noDataRead
		r.Takeaway = "No output tokens in this window."
		return r
	}
	var reasoning, reportingOutput int64
	for i := range in.Usage {
		u := &in.Usage[i]
		if !reportsReasoning(u.Tool) {
			continue
		}
		reasoning += u.Reasoning
		reportingOutput += u.Out
	}
	if reportingOutput == 0 {
		r.Read = noDataRead
		r.Figures = []Figure{{Label: "reporting coverage", Value: "0%", Note: "no tool here reports reasoning"}}
		r.Takeaway = "None of your tools reported reasoning tokens this window -- Claude Code doesn't surface them."
		r.Caveats = []string{"Only Codex and Gemini CLI report reasoning tokens today; Claude Code doesn't."}
		return r
	}

	// Share is of output from tools that actually report reasoning, so a Claude-heavy
	// window (Claude doesn't surface it) isn't diluted to a meaningless near-zero.
	share := fracOf(reasoning, reportingOutput)
	coverage := fracOf(reportingOutput, in.Totals.Output)

	r.Read = readFor(share < reasoningWatchShare, "Lean")
	r.Purity = clamp01(1 - share)
	r.Figures = []Figure{
		{Label: "reasoning share", Value: honestPercent(share), Note: "of reporting output"},
		{Label: "reasoning tokens", Value: compactCount(reasoning)},
		{Label: "reporting coverage", Value: honestPercent(coverage), Note: "output from tools that report it"},
	}
	r.Takeaway = reasoningTakeaway(share)
	r.Caveats = []string{
		"Only Codex and Gemini CLI report reasoning tokens; Claude Code doesn't, so this covers part of your output.",
		"Reasoning is a subset of output (already billed there); this is a composition signal, not extra cost.",
	}
	return r
}

// reportsReasoning reports whether tool's parser surfaces reasoning tokens today: Codex
// (reasoning_output_tokens) and Gemini CLI (thoughts). Claude Code does not.
func reportsReasoning(tool string) bool {
	return tool == "codex" || tool == "gemini-cli"
}

func reasoningTakeaway(share float64) string {
	if share >= reasoningWatchShare {
		return "A large share of reported output is reasoning -- worth checking whether deep thinking is going to shallow tasks."
	}
	return "Reasoning is a modest share of reported output -- the thinking budget looks proportionate."
}
