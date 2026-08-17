package share

import (
	"fmt"
	"sort"
	"strings"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/store"
)

// Hashtag names what people are joining rather than the tool they installed, because a
// tag that reads as a vendor's name is a tag nobody re-uses. It is on every post and on
// the artwork: a caption is dropped when an image is re-shared, and the image is the part
// that travels.
const Hashtag = "#ProofOfPrompt"

// InstallCommand and ShareCommand are the two lines that turn a reader into the next
// poster. They are in the post itself rather than behind a link: "two commands" is only
// persuasive when both are visible and countable.
const (
	InstallCommand = "brew install assaio/tap/assaio-agent"
	ShareCommand   = "assaio-agent share"
)

// post writes the text that travels with the image. Two halves, because a post that only
// flatters is read as an advert and a post that only confesses is read as a complaint: the
// fame half is what the window produced, the shame half is the same window's own reserve.
// It closes on the two commands and a question, which together are the whole distribution
// mechanism -- a reader who is asked for their number, and shown exactly how much work
// answering costs, is a reader who makes the next card.
func post(a *Assay, f *facts) string {
	var b strings.Builder
	fmt.Fprintf(&b, "My %s of AI, measured rather than guessed.\n\n", a.Window)
	fmt.Fprintf(&b, "FAME  · %s\n", fameLine(f))
	fmt.Fprintf(&b, "SHAME · %s\n", shameLine(a, f))
	// The hook only earns a line here when it says something the two halves above did not.
	// A scale or money hook repeats the fame line word for word, and a post that states its
	// own headline twice reads as padding.
	if a.Hook.Family != "scale" && a.Hook.Family != "money" && a.Hook.Line != "" {
		fmt.Fprintf(&b, "\n%s\n", a.Hook.Line)
	}
	b.WriteString("\nTwo commands and you have yours:\n")
	fmt.Fprintf(&b, "  %s\n  %s\n", InstallCommand, ShareCommand)
	b.WriteString("\nCounted locally. Nothing uploaded, no account, no repo names.\n\n")
	fmt.Fprintf(&b, "%s — what are your numbers of fame or shame? ;)", Hashtag)
	return b.String()
}

func fameLine(f *facts) string {
	parts := []string{
		humanize.Count(f.tokens) + " tokens",
		humanize.Int(f.lines) + " lines",
	}
	if f.perHundred != "" {
		parts = append(parts, f.perHundred+" per 100")
	}
	return strings.Join(parts, ", ")
}

// shameLine quotes the profile's own reserve, so the post cannot praise a window the card
// is simultaneously flagging.
func shameLine(a *Assay, f *facts) string {
	if a.Profile.Line != "" {
		return a.Profile.Line
	}
	if f.conversational.ok {
		return f.conversational.text() + " of my sessions never touched a file"
	}
	return "the window is too young to have a verdict yet"
}

// fine is the card's small print. Every line is a limit on what the numbers above may be
// read as, and the list is assembled from what this window actually did -- an unpriced
// window says so, a sample run says so.
func fine(f *facts, v verdicts, sample bool) []string {
	out := []string{
		"Activity and output layers. Nothing here claims the code was good, only that it was written.",
		"Cost is an estimate at public pay-as-you-go API prices, not an actual bill; flat plans bill differently.",
		"Prompts and source code were not collected. No repository, branch, path, skill or sub-agent is named.",
	}
	if sample {
		out = append([]string{"Sample data from `assaio-agent demo` — not a real machine's usage."}, out...)
	}
	if s := v.value("coverage", "activity coverage"); s != "" {
		out = append(out, fmt.Sprintf("Activity coverage %s, priced coverage %s, over %d active days.", s, v.value("coverage", "priced coverage"), f.activeDays))
	}
	return out
}

// publishableTool keeps a built-in tool's name and collapses everything else. An out-of-tree
// parser is stored as "plugin:<name>" where the name is a string from the user's own config
// (ADR 0003) -- "plugin:acme-internal-billing" is exactly the kind of label the redaction rule
// exists to keep off a public card, and the vendor-fact argument that licenses naming
// claude-code does not reach it.
func publishableTool(tool string) string {
	switch tool {
	case "claude-code", "codex", "gemini-cli", "copilot-cli", "cline":
		return tool
	default:
		return "a plugin source"
	}
}

// toolSlices ranks the window's tools by tokens. Naming a tool is deliberate and is the
// one dimension besides models the redaction rule permits: which coding agent ran is a
// public fact about a vendor, not about the person who ran it.
func toolSlices(rows []store.UsageRow) []Slice {
	totals := map[string]int64{}
	var all int64
	for i := range rows {
		r := &rows[i]
		n := r.In + r.Out + r.CacheRead + r.CacheWrite
		totals[publishableTool(r.Tool)] += n
		all += n
	}
	if all == 0 {
		return nil
	}
	names := make([]string, 0, len(totals))
	for name := range totals {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if totals[names[i]] != totals[names[j]] {
			return totals[names[i]] > totals[names[j]]
		}
		return names[i] < names[j]
	})
	out := make([]Slice, 0, len(names))
	for i, name := range names {
		hue := 4
		if i < 4 {
			hue = i
		}
		out = append(out, Slice{
			Label: name,
			Value: humanize.PercentAt(float64(totals[name])/float64(all), 1),
			Frac:  float64(totals[name]) / float64(all),
			Hue:   hue,
		})
	}
	return out
}
