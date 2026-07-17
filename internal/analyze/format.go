package analyze

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// noDataRead is the shared Read for a validator whose window had no data to analyze.
var noDataRead = Read{Key: "neutral", Label: "—"}

// readFor returns the favorable Read (Key "good", favorableLabel upper-cased) when ok,
// else the shared "WATCH" fallback (Key "watch") every validator uses for its
// non-favorable state.
func readFor(ok bool, favorableLabel string) Read {
	if ok {
		return Read{Key: "good", Label: strings.ToUpper(favorableLabel)}
	}
	return Read{Key: "watch", Label: "WATCH"}
}

// RenderResultText writes r as the CLI's clean text block: a header line, a "? "
// how-to-read line, each Figure, each Bar (or an honest "none in this window" when Bars
// was populated but empty), any caveats, and the takeaway. Every validator renders
// through this one function, so the text report and a future HTML dashboard read the
// same Result the same way.
//
//nolint:gocritic // Result is caller-constructed and rendered once per validator per CLI run; by-value matches Validator.Analyze's own return and keeps this the simple entry point it flows into directly.
func RenderResultText(w io.Writer, r Result) error {
	if _, err := fmt.Fprintf(w, "%s · %s  [%s]\n", strings.ToUpper(r.Name), r.Title, r.Read.Label); err != nil {
		return err
	}
	if r.HowToRead != "" {
		if _, err := fmt.Fprintf(w, "  ? %s\n", r.HowToRead); err != nil {
			return err
		}
	}
	for _, f := range r.Figures {
		if err := writeFigure(w, f); err != nil {
			return err
		}
	}
	if err := writeBars(w, r.Bars); err != nil {
		return err
	}
	for _, c := range r.Caveats {
		if _, err := fmt.Fprintf(w, "  %s\n", c); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "  Takeaway: %s\n", r.Takeaway); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w)
	return err
}

// writeFigure writes one stat line: "  label: value" plus an optional "(note)".
func writeFigure(w io.Writer, f Figure) error {
	if f.Note == "" {
		_, err := fmt.Fprintf(w, "  %s: %s\n", f.Label, f.Value)
		return err
	}
	_, err := fmt.Fprintf(w, "  %s: %s (%s)\n", f.Label, f.Value, f.Note)
	return err
}

// barWidth is the textual bar's full-scale character count.
const barWidth = 20

// writeBars writes each Bar as "  label: value  [####----]". bars == nil means the
// Result carries no Bars section at all, so nothing is written; a non-nil empty bars
// (a section that legitimately has no rows right now) prints an honest one-liner instead
// of silently showing nothing.
func writeBars(w io.Writer, bars []Bar) error {
	if bars == nil {
		return nil
	}
	if len(bars) == 0 {
		_, err := fmt.Fprintln(w, "  (none in this window)")
		return err
	}
	for _, b := range bars {
		if err := writeBar(w, b); err != nil {
			return err
		}
	}
	return nil
}

func writeBar(w io.Writer, b Bar) error {
	filled := int(b.Frac*barWidth + 0.5)
	if filled > barWidth {
		filled = barWidth
	} else if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
	_, err := fmt.Fprintf(w, "  %s: %s  [%s]\n", b.Label, b.Value, bar)
	return err
}

// formatPercent renders a 0-1 share as a percentage at the given decimal precision,
// e.g. formatPercent(0.999, 1) -> "99.9%".
func formatPercent(share float64, decimals int) string {
	return strconv.FormatFloat(share*100, 'f', decimals, 64) + "%"
}

// shareOrDash divides num by den as a percentage at the given decimal precision, "—"
// when den is zero -- never a divide-by-zero substitute for a real ratio.
func shareOrDash(num, den int64, decimals int) string {
	if den == 0 {
		return "—"
	}
	return formatPercent(float64(num)/float64(den), decimals)
}

// perActiveDay divides n by days to one decimal place, "—" when days is zero.
func perActiveDay(n, days int64) string {
	if days == 0 {
		return "—"
	}
	return strconv.FormatFloat(float64(n)/float64(days), 'f', 1, 64)
}

// groupLabel substitutes "(unknown)" for an empty group name, mirroring the status
// dashboard's convention for an unset dimension value.
func groupLabel(name string) string {
	if name == "" {
		return "(unknown)"
	}
	return name
}

// fracOf divides n by maxN as a 0..1 Bar.Frac, 0 when maxN is zero.
func fracOf(n, maxN int64) float64 {
	if maxN == 0 {
		return 0
	}
	return float64(n) / float64(maxN)
}

// clamp01 constrains x to [0,1], the valid range for Result.Purity.
func clamp01(x float64) float64 {
	switch {
	case x < 0:
		return 0
	case x > 1:
		return 1
	default:
		return x
	}
}
