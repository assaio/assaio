package cli

import (
	"fmt"
	"strings"

	"github.com/assaio/assaio/internal/parser"
)

// depthLegend explains the tiers once, so the per-source labels above it mean something
// without a trip to the docs. activity-only is named even though only one source is in it: a
// reader who skips the gap lines below would otherwise take an unfamiliar tier for a lesser
// grade of the same thing, when it is the opposite half of the measurement.
const depthLegend = "  deep = tokens + per-turn activity + attribution labels; standard = usage with the gaps below;\n" +
	"  activity-only = sessions and what was done in them, and no token or cost figure at all"

// doctorDepthSection reports what each detected source can actually tell you. "Supports
// Gemini CLI" is misleading when it means tokens but no edits, so the tier comes with the
// specific gaps behind it -- and only for sources present on this machine, since spelling
// out what an uninstalled tool cannot do is the noise that makes a reader skip the section.
func doctorDepthSection(scans []sourceScan) string {
	var tiers []string
	var gaps []string
	for i := range scans {
		if scans[i].found == 0 {
			continue
		}
		d, ok := parser.DepthOf(scans[i].tool)
		if !ok {
			continue
		}
		tiers = append(tiers, d.Tool+" "+d.Tier)
		for _, g := range d.Gaps {
			gaps = append(gaps, fmt.Sprintf("  %s — %s\n", d.Tool, g))
		}
	}
	if len(tiers) == 0 {
		return "depth:        no source detected yet\n"
	}
	return fmt.Sprintf("depth:        %s\n%s\n%s",
		strings.Join(tiers, " · "), depthLegend, strings.Join(gaps, ""))
}
