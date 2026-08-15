package store

import (
	"regexp"
	"testing"
)

// derivedColumns are the stored figures assaio computes by a rule of its own rather than
// reading off a vendor's log: which lines are edits, which removal undoes an earlier addition,
// which position a step holds, which four token fields a step's total sums. A rule assaio wrote
// is a rule assaio can get wrong, so a restate must assign them -- MAX pins every stored row at
// the old figure forever and a correction reaches nothing (B116). A vendor's own token count is
// the opposite case and may take MAX, which repairs a read of a file still being written.
var derivedColumns = []string{
	"granularity",
	"lines_added", "lines_removed", "rework_lines",
	"edits", "tool_calls", "rejected", "compactions",
	"tool_reads", "tool_searches", "tool_commands", "tool_writes", "tool_other", "tool_errors",
	"ordinal", "target_ref", "tokens", "outcome",
}

// restateStatements is every UPDATE a re-read of a file this store owns can run. Hand-maintained,
// which is how `tokens = MAX` shipped in a new table with nothing looking -- so a statement added
// to the package belongs here in the same commit.
var restateStatements = map[string]string{
	"restateActivitySQL": restateActivitySQL,
	"restateSignalsSQL":  restateSignalsSQL,
	"restateStepSQL":     restateStepSQL,
}

// TestNoRestateMaxesAnAssaioDerivedFigure is the guard the two halves of B116 were found
// without: v0.13 caught rework_lines, v0.14 four more counts, and v0.20 shipped `tokens = MAX`
// in a brand-new table anyway, because nothing looked. It reads the shipped SQL rather than
// behaviour, so a column added to a restate is covered the moment it is named.
func TestNoRestateMaxesAnAssaioDerivedFigure(t *testing.T) {
	for _, col := range derivedColumns {
		// Every shape SQLite accepts that keeps the stored value, not just the one that shipped:
		// `MAX(`, `max (`, `IIF(` and a fill-only `CASE WHEN` all pin a row, and the CASE idiom
		// was already in use one line above `tokens` -- so it is the one a contributor reaches
		// for next.
		pinned := regexp.MustCompile(`(?i)(^|[\s,(])` + regexp.QuoteMeta(col) + `\s*=\s*(max\s*\(|iif\s*\(|case\s)`)
		assigned := regexp.MustCompile(`(?i)(^|[\s,(])` + regexp.QuoteMeta(col) + `\s*=\s*`)
		seen := false
		for name, body := range restateStatements {
			if pinned.MatchString(body) {
				t.Errorf("%s keeps the stored %s rather than assigning it; %s is assaio-derived, so a corrected rule could never reach a stored row", name, col, col)
			}
			if assigned.MatchString(body) {
				seen = true
			}
		}
		if !seen {
			t.Errorf("%s is listed as derived but no restate sets it; the list and the SQL have drifted", col)
		}
	}
}
