package recommend

import (
	"fmt"
	"io"
	"strings"
)

// RenderText writes records as the CLI's block per recommendation. Every line projects a field
// of the Record: there is no sentence here that the record does not hold, which is what makes
// the advice auditable rather than persuasive -- and what stops a later renderer quietly
// gaining a claim the contract never carried.
func RenderText(w io.Writer, records []Record) error {
	if len(records) == 0 {
		_, err := fmt.Fprint(w, abstained)
		return err
	}
	for i := range records {
		if err := renderOne(w, &records[i]); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, footer)
	return err
}

// abstained is what an empty list means, said plainly. "No recommendations" reads as a clean
// bill of health and is not one: assaio abstains whenever coverage, confidence or a baseline is
// too thin to build an action on, which on a young store is most of the time.
const abstained = `No recommendation from this window.

  That is an abstention, not a clean bill. A recommendation is only emitted where the evidence
  under it is strong enough that acting on it is not acting on noise -- so a thin window, a
  low-confidence verdict, or a metric whose declared inputs are missing produces nothing at all
  rather than a plausible suggestion. Run 'assaio-agent analyze' to see what the window does
  support.
`

const footer = `
Each of these is an experiment, not an instruction: it names what it rests on, what it would
cost you to undo, when to look again, and the figure that would show whether it worked. assaio
has no counterfactual and predicts no number -- it can only tell you afterwards what moved.
`

func renderOne(w io.Writer, r *Record) error {
	if _, err := fmt.Fprintf(w, "%s  [%s]\n  %s\n", strings.ToUpper(r.Family), strings.ToUpper(string(r.Status)), r.Title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  scope: %s · confidence: %s\n", r.Scope, r.Confidence); err != nil {
		return err
	}
	for _, section := range []struct {
		label string
		lines []string
	}{
		{"evidence", r.Evidence},
		{"action", []string{r.Action}},
		{"prerequisites", r.Prerequisites},
		{"expected effect", []string{r.Effect}},
		{"risks", r.Risks},
		{"rollback", []string{r.Rollback}},
		{"review after", []string{r.ReviewWindow}},
		{"follow-up", []string{r.FollowUp}},
	} {
		if err := renderSection(w, section.label, section.lines); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

// renderSection prints one labelled block, and nothing at all when it is empty -- an absent
// section is the honest form of "this record does not carry one", where an empty heading reads
// as a field somebody forgot.
func renderSection(w io.Writer, label string, lines []string) error {
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		head := label + ":"
		if i > 0 {
			head = strings.Repeat(" ", len(label)+1)
		}
		if _, err := fmt.Fprintf(w, "  %-16s %s\n", head, line); err != nil {
			return err
		}
	}
	return nil
}
