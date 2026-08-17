package share

import (
	"fmt"
	"io"
	"strings"
)

// lines is the errWriter pattern: many sequential writes, one error reported. The text
// card emits dozens of lines and a check after each would bury the layout it exists to
// make readable.
type lines struct {
	w   io.Writer
	err error
}

func (l *lines) printf(format string, a ...any) {
	if l.err != nil {
		return
	}
	_, l.err = fmt.Fprintf(l.w, format, a...)
}

// textWidth is the inner width of the boxed card, chosen to sit inside an 80-column
// terminal with room for a quote marker when someone pastes it into a thread.
const textWidth = 66

// writeText renders the assay as a boxed block. Padding is computed rather than typed, so
// the frame stays square whatever a window's own numbers turn out to be.
func writeText(out io.Writer, a *Assay) error {
	w := &lines{w: out}
	head0 := fmt.Sprintf("ASSAY · %s · %d active days", a.Window, a.Days)
	w.printf("%s\n", capLine(head0, '┌', '┐'))
	row(w, "")
	for _, line := range wrap(a.Hook.Line, textWidth-4) {
		row(w, "  "+line)
	}
	if a.Hook.Sub != "" {
		row(w, "  "+a.Hook.Sub)
	}
	row(w, "")
	for _, m := range a.Models {
		row(w, fmt.Sprintf("  %-26s %s %6s", trim(m.Label, 26), bar(m.Frac, 18), m.Value))
	}
	row(w, "")
	row(w, "  "+a.Archetype.Name)
	for _, line := range wrap(a.Archetype.Flavor, textWidth-4) {
		row(w, "  "+line)
	}
	row(w, "")
	for _, s := range a.Ledger {
		row(w, fmt.Sprintf("  %-28s %s", s.Label, s.Value))
		if s.Note != "" {
			for _, line := range wrap("    "+s.Note, textWidth-4) {
				row(w, "  "+line)
			}
		}
	}
	if len(a.Profile.Axes) > 0 {
		row(w, "")
		for _, ax := range a.Profile.Axes {
			mark := " "
			if ax.Reserve {
				mark = "*"
			}
			row(w, fmt.Sprintf("%s %-22s %s", mark, ax.Left+" — "+ax.Right, ax.Value))
		}
	}
	if a.Profile.Line != "" {
		row(w, "")
		for _, line := range wrap("* reserve · "+a.Profile.Line, textWidth-4) {
			row(w, "  "+line)
		}
	}
	row(w, "")
	for _, line := range wrap(a.Limit, textWidth-4) {
		row(w, "  "+line)
	}
	row(w, "")
	w.printf("%s\n", capLine("make yours: assaio.dev", '└', '┘'))
	for _, line := range a.Fine {
		for _, part := range wrap(line, textWidth) {
			w.printf("  %s\n", part)
		}
	}
	w.printf("\n%s\n", a.Post)
	return w.err
}

func row(w *lines, s string) {
	w.printf("│%s│\n", pad(s, textWidth))
}

func capLine(title string, left, right rune) string {
	head := fmt.Sprintf("%c─ %s ", left, title)
	fill := textWidth + 2 - runeLen(head) - 1
	if fill < 0 {
		fill = 0
	}
	return head + strings.Repeat("─", fill) + string(right)
}

func bar(frac float64, width int) string {
	n := int(frac*float64(width) + 0.5)
	if n < 1 && frac > 0 {
		n = 1
	}
	if n > width {
		n = width
	}
	return strings.Repeat("█", n) + strings.Repeat("─", width-n)
}

func pad(s string, width int) string {
	if n := width - runeLen(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return trim(s, width)
}

func trim(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return string(r[:width-1]) + "…"
}

func runeLen(s string) int { return len([]rune(s)) }

// wrap breaks s on spaces at width, never mid-word, so a long honesty note stays readable
// instead of being truncated into a different claim.
func wrap(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var out []string
	line := words[0]
	for _, word := range words[1:] {
		if runeLen(line)+1+runeLen(word) > width {
			out = append(out, line)
			line = word
			continue
		}
		line += " " + word
	}
	return append(out, line)
}
