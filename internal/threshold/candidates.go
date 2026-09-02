package threshold

import "time"

func init() {
	Register(&reworkAgainstGitClearChurn)
	Register(&survivalAgainstGitClearChurn)
}

// gitClearChurn is the figure every discussion of AI code quality eventually quotes, and the
// one a reader will hold assaio's rates up against whether or not assaio invites them to.
//
// ValidUntil is a year past Checked because GitClear restates this series annually: a figure
// that outlives the next edition is quoting a number the source itself has already replaced.
var gitClearChurn = Citation{
	Value: "~3.3% of changed lines before AI assistants, rising to 5.7% in 2024",
	Definition: "code authored, pushed to the git repo, and then reverted or substantially " +
		"revised within the subsequent two weeks, as a share of all changed lines",
	Source: "GitClear, AI Copilot Code Quality: Evaluating 2024's Increased Defect Rate",
	URL:    "https://www.gitclear.com/ai_assistant_code_quality_2025_research",
	Population: "211 million changed lines analysed by GitClear, January 2020 through December " +
		"2024. The 7.1% usually quoted beside those figures is the report's projection for 2025, " +
		"past the end of its own observation period, and the linked page does not print the " +
		"year-by-year series.",
	Measured:   date(2024, time.December, 31),
	Checked:    date(2026, time.September, 2),
	ValidUntil: date(2027, time.September, 2),
}

// reworkAgainstGitClearChurn is the pairing a reader makes unprompted: assaio prints a rework
// percentage, GitClear prints a churn percentage, and subtracting them reads as a grade. Every
// one of the five properties is a different quantity, so the register's answer is to name the
// study rather than to leave a silence a reader fills with it.
var reworkAgainstGitClearChurn = Candidate{
	Metric:   "rework",
	Citation: gitClearChurn,
	Differs: "GitClear pooled changed lines across many repositories and every author in them; " +
		"assaio's rate is one developer's own AI sessions on their own projects, and never leaves " +
		"the transcript to ask whether the code was pushed at all",
	Fit: []Comparison{
		{
			Property: Numerator,
			Cited:    "lines reverted or substantially revised within two weeks of being pushed",
			Assaio:   "AI-added lines removed again inside the same transcript and file",
		},
		{
			Property: Denominator,
			Cited:    "all changed lines, whoever wrote them",
			Assaio:   "AI-added code lines only",
		},
		{
			Property: Window,
			Cited:    "the two weeks after each push, equal for every line",
			Assaio:   "one session, however long or short that session ran",
		},
		{
			Property: DataSource,
			Cited:    "git history, so only code that was committed and pushed counts",
			Assaio:   "the tool's own transcript, where code that was never committed counts too",
		},
		{
			Property: Layer,
			Cited:    "outcome -- whether code held up after it landed",
			Assaio:   "output -- a share of what was produced, with no claim about what landed",
		},
	},
}

// survivalAgainstGitClearChurn is the nearer pairing of the two and still not close enough: it
// agrees with the study on where the evidence comes from and on what kind of claim it is, and
// disagrees on all three of what is counted, what it is counted out of, and over how long.
//
// The window row is the disqualifying one rather than a technicality. Survival is monotonic in
// commit age, so the same repository reads near 100% over a week and far lower over a year;
// AGENTS.md requires an age-matched comparison before any AI-versus-human claim, and a fixed
// two-week horizon cannot be age-matched against a window of commits of every age.
var survivalAgainstGitClearChurn = Candidate{
	Metric:   "survival",
	Citation: gitClearChurn,
	Differs: "GitClear pooled many repositories at a fixed two-week horizon per line; assaio " +
		"blames one repository's working tree over commits of every age in whatever window the " +
		"caller asked for, and cannot say which of those lines an AI wrote",
	Fit: []Comparison{
		{
			Property: Numerator,
			Cited:    "lines reverted or substantially revised within two weeks of being pushed",
			Assaio: "lines HEAD's blame still attributes to the window's commits -- a line that " +
				"was moved, reformatted or resolved in a merge stops being attributed without " +
				"having been revised in substance",
		},
		{
			Property: Denominator,
			Cited:    "all changed lines in the analysed repositories",
			Assaio: "lines the window's commits added in one repository, with modifications and " +
				"deletions outside it and no attribution to AI or human",
		},
		{
			Property: Window,
			Cited:    "the two weeks after each push, equal for every line",
			Assaio: "from each commit until now, so the oldest commit in the window has had years " +
				"to be rewritten and the newest has had hours",
		},
		{
			Property: DataSource,
			Cited:    "git history",
			Assaio:   "git history, read by internal/vcs and blamed against the working tree",
			Same:     true,
		},
		{
			Property: Layer,
			Cited:    "outcome -- whether code held up after it landed",
			Assaio:   "outcome -- whether committed lines are still present in HEAD",
			Same:     true,
		},
	},
}

// date is a citation's dates written the way a reader checks them: a day, in UTC, with no time
// of day pretending to a precision a published report does not have.
func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
