package reprice

import (
	"fmt"
	"io"
	"strings"

	"github.com/assaio/assaio/internal/humanize"
)

const (
	title     = "REPRICE · What this window costs against another price table"
	howToRead = "Re-prices turns assaio already read against another entry in the same price table. The plan half asks " +
		"whether a flat price is worth this volume; the model half asks what these same tokens cost elsewhere. Both are " +
		"arithmetic over stored events, not a prediction of what another model or another plan would have done."
)

// RenderText writes w as the CLI block. Every line projects a field of the Window -- the
// assumptions and the refusals included -- so a renderer cannot grow a claim the computation
// does not hold, which is the property that makes an arbitrage figure auditable rather than
// persuasive.
func RenderText(out io.Writer, w *Window) error {
	for _, write := range []func(io.Writer, *Window) error{
		writeHeader, writeBasis, writeRoutes, writePlans, writeDisclosure,
	} {
		if err := write(out, w); err != nil {
			return err
		}
	}
	return nil
}

func writeHeader(out io.Writer, w *Window) error {
	_, err := fmt.Fprintf(out, "%s  (%s)\n  ? %s\n\n", title, w.Layer, howToRead)
	return err
}

func writeBasis(out io.Writer, w *Window) error {
	b := &w.Basis
	if err := heading(out, "observed basis"); err != nil {
		return err
	}
	if err := field(out, "window", windowSpan(b), "the span every projection below rests on"); err != nil {
		return err
	}
	if !b.Priced {
		if err := field(out, "priced cost", "—", "nothing in this window carries a known price, so there is nothing to re-price"); err != nil {
			return err
		}
		return blank(out)
	}
	rows := [][3]string{
		{"priced cost", humanize.USDCompact(b.Cost), "the observed window at the vendored public API table"},
		{"monthly pace", humanize.USDCompact(b.Monthly) + "/mo", "that cost projected over " + humanize.Days(b.Days)},
	}
	if len(b.Premium.Models) > 0 {
		rows = append(rows, [3]string{
			"premium slice",
			humanize.USDCompact(b.Premium.Cost) + " (" + humanize.Percent(b.PremiumShare()) + ")",
			names(b.Premium.Models),
		})
	}
	if b.Unpriced.Missing() {
		rows = append(rows, [3]string{"unpriced", humanize.PercentAt(b.Unpriced.Share, 1) + " of tokens", names(b.Unpriced.Models)})
	}
	for _, r := range rows {
		if err := field(out, r[0], r[1], r[2]); err != nil {
			return err
		}
	}
	return blank(out)
}

func writeRoutes(out io.Writer, w *Window) error {
	if err := heading(out, "the same premium turns, priced against another model"); err != nil {
		return err
	}
	if len(w.Routes) == 0 {
		if err := field(out, "none", "", routeAbsence(w)); err != nil {
			return err
		}
		return writeUnpriceable(out, w)
	}
	if err := routeRow(out, "model", "slice", "whole window", "vs observed"); err != nil {
		return err
	}
	for i := range w.Routes {
		r := &w.Routes[i]
		// The columns state the move as a change in what the window cost, so the sign is
		// inverted from Delta: a delta that saves money makes the window smaller.
		if err := routeRow(out, r.Target, humanize.USDCompact(r.Premium), humanize.USDCompact(r.Window),
			signedMoney(-r.Delta)+" ("+signedPercent(-r.Share)+")"); err != nil {
			return err
		}
	}
	return writeUnpriceable(out, w)
}

// writeUnpriceable names the targets that never reached the table, then closes the section. It
// runs whether or not a row was printed, because the row that was printed is exactly what makes
// a silent skip unreadable.
func writeUnpriceable(out io.Writer, w *Window) error {
	if len(w.Unpriceable) > 0 {
		// Not a row in the table above and not a labelled field either: it is neither a route nor
		// a figure, and borrowing the columns of one would read as the other.
		if err := writeLine(out, "    not priced: "+names(w.Unpriceable)+
			" — the vendored price table carries no rate for it, so there is nothing to re-price onto"); err != nil {
			return err
		}
	}
	return blank(out)
}

// routeAbsence says which of the four reasons left the table empty. They send a reader to four
// different places, so one "no routes" line would be the least useful of the four.
func routeAbsence(w *Window) string {
	switch {
	case !w.Basis.Priced:
		return "nothing in this window carries a known price"
	case len(w.Basis.Premium.Models) == 0:
		return "no token in this window ran on a model the price table places in the premium tier"
	case len(w.Unpriceable) > 0:
		return "this window ran no other priced model, and the price table carries no rate for the target(s) named below"
	default:
		return "this window ran no other priced model to re-price onto -- name one with --against <model>"
	}
}

func writePlans(out io.Writer, w *Window) error {
	if err := heading(out, "a flat plan against that monthly pace"); err != nil {
		return err
	}
	if len(w.Plans) == 0 {
		if err := field(out, "none", "", "no plan price to compare against: set pricing.monthly_subscription_cost, "+
			"or pass --plan \"Max 20x=200\" with the figure from your vendor's page"); err != nil {
			return err
		}
		return blank(out)
	}
	for i := range w.Plans {
		p := &w.Plans[i]
		if err := field(out, p.Name, "$"+humanize.USD(p.Monthly)+"/mo", multipleNote(p.Multiple)+" · "+p.Source); err != nil {
			return err
		}
	}
	return blank(out)
}

func writeDisclosure(out io.Writer, w *Window) error {
	if err := writeList(out, "assumes", w.Assumptions); err != nil {
		return err
	}
	return writeList(out, "does not claim", w.Refusals)
}

func writeList(out io.Writer, label string, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	if err := heading(out, label); err != nil {
		return err
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(out, "    - %s\n", line); err != nil {
			return err
		}
	}
	return blank(out)
}

// heading, field and routeRow are the block's whole shape, stated once so no section can
// drift into a layout of its own.
func heading(out io.Writer, s string) error {
	_, err := fmt.Fprintf(out, "  %s\n", s)
	return err
}

func field(out io.Writer, label, value, note string) error {
	return writeLine(out, fmt.Sprintf("    %-16s %-18s %s", label, value, note))
}

func routeRow(out io.Writer, model, slice, window, delta string) error {
	return writeLine(out, fmt.Sprintf("    %-28s %10s %14s   %s", model, slice, window, delta))
}

func writeLine(out io.Writer, line string) error {
	_, err := fmt.Fprintln(out, strings.TrimRight(line, " "))
	return err
}

func blank(out io.Writer) error {
	_, err := fmt.Fprintln(out)
	return err
}
