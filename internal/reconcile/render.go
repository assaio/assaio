package reconcile

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/assaio/assaio/internal/humanize"
)

// RenderJSON writes r as indented JSON.
func RenderJSON(w io.Writer, r *Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// RenderText writes the reconciliation in the order it has to be read: what was compared,
// how far the two windows failed to line up, then the delta, then what is left unexplained,
// and last what none of it can answer.
func RenderText(w io.Writer, r *Result) error {
	p := &printer{w: w}
	p.line("Reconciliation — a vendor export against assaio's own estimate")
	p.line("  export      %s", r.Source)
	p.line("  read as     %s", bindingLine(&r.Binding))
	p.line("  rows        %d read, %d skipped", r.Rows-r.Skipped, r.Skipped)

	p.blank()
	p.line("Scope — computed first, because an export window is not the queried window")
	p.line("  export covers    %s .. %s", orDash(r.Scope.ExportFirst), orDash(r.Scope.ExportLast))
	p.line("  window queried   %s .. %s", r.Scope.WindowFirst, r.Scope.WindowLast)
	p.line("  compared over    %d days", r.Scope.OverlapDays)
	if !r.Scope.Aligned() {
		p.line("  excluded         %d days only in the export (%s), %d days only in the window (%s)",
			r.Scope.ExportOnlyDays, money(r.Scope.ExportCostOutside),
			r.Scope.WindowOnlyDays, money(r.Scope.EstimateCostOutside))
	}

	p.blank()
	p.line("Over those %d days", r.Scope.OverlapDays)
	p.line("  vendor billed    %s", money(r.ExportCost))
	p.line("  assaio estimated %s%s", money(r.Estimate.Priced), bandSuffix(&r.Estimate))
	p.line("  delta            %s   (%s)", signed(r.Delta), direction(r.Delta))
	if r.Estimate.Banded() {
		p.line("  band             %s", bandVerdict(r))
	}

	p.blank()
	if len(r.Causes) == 0 {
		p.line("Named causes — none. Nothing in this comparison has evidence behind it,")
		p.line("so the whole delta is unexplained.")
	} else {
		p.line("Named causes — only where there is evidence")
		for _, c := range r.Causes {
			p.line("  %s  %s", pad(signed(c.Amount), 12), c.Name)
			p.line("  %s  %s", strings.Repeat(" ", 12), c.Evidence)
		}
	}

	p.blank()
	p.line("Unexplained delta   %s", signed(r.Unexplained))
	p.line("  This is the output, not an error. assaio does not adjust either side to close it.")

	if len(r.Limits) > 0 {
		p.blank()
		p.line("What this run could not check")
		for _, l := range r.Limits {
			p.wrapped("  - ", l)
		}
	}
	p.blank()
	p.line("What no export can check")
	for _, l := range r.Refusals {
		p.wrapped("  - ", l)
	}
	return p.err
}

type printer struct {
	w   io.Writer
	err error
}

func (p *printer) line(format string, args ...any) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintf(p.w, format+"\n", args...)
}

func (p *printer) blank() { p.line("") }

// wrapped prints a long note at a readable measure, indenting continuations under the first.
func (p *printer) wrapped(prefix, text string) {
	const width = 84
	indent := strings.Repeat(" ", len(prefix))
	line := prefix
	for _, word := range strings.Fields(text) {
		if len(line) > len(prefix) && len(line)+1+len(word) > width {
			p.line("%s", line)
			line = indent
		}
		if line != prefix && line != indent {
			line += " "
		}
		line += word
	}
	if strings.TrimSpace(line) != "" {
		p.line("%s", line)
	}
}

func bindingLine(b *Binding) string {
	parts := []string{"day=" + b.Day, "cost=" + b.Cost}
	for _, kv := range [][2]string{{"model", b.Model}, {"tokens", b.Tokens}, {"currency", b.Currency}} {
		if kv[1] != "" {
			parts = append(parts, kv[0]+"="+kv[1])
		}
	}
	return strings.Join(parts, "  ")
}

func bandSuffix(e *Estimate) string {
	switch {
	case e.Banded():
		return fmt.Sprintf(" .. %s   (upper bound extrapolates unpriced usage at this window's own $/token)", money(e.Upper()))
	case e.UnpricedRows > 0:
		return fmt.Sprintf("   (a point: %d unpriced %s, none carrying tokens to extrapolate from)",
			e.UnpricedRows, plural(e.UnpricedRows, "row", "rows"))
	default:
		return "   (a point: every row in the window carried a known price)"
	}
}

func bandVerdict(r *Result) string {
	if r.InBand() {
		return "the vendor's total lands inside the range the estimate could honestly claim"
	}
	if r.ExportCost > r.Estimate.Upper() {
		return "the vendor billed " + money(r.ExportCost-r.Estimate.Upper()) + " above the top of the band"
	}
	return "the vendor billed " + money(r.Estimate.Priced-r.ExportCost) + " below the priced estimate"
}

func direction(d float64) string {
	switch {
	case d > 0:
		return "the vendor billed more than the local logs account for"
	case d < 0:
		return "the local logs account for more than the vendor billed"
	default:
		return "the two sides land on the same total"
	}
}

func money(v float64) string { return "$" + humanize.USDCell(v) }

func signed(v float64) string {
	if v > 0 {
		return "+" + money(v)
	}
	if v < 0 {
		return "-" + money(-v)
	}
	return money(0)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat(" ", n-len(s)) + s
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
