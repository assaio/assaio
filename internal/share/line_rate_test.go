package share

import (
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/report"
)

// TestTheRatioTravelsOnlyWhereItsNoteCan: the hook line and the post cannot carry a note, so a
// $/100-lines figure that leaves usage out stays in the ledger, where its note says how much.
// A window whose sources record no lines never prints "0 lines" anywhere.
func TestTheRatioTravelsOnlyWhereItsNoteCan(t *testing.T) {
	full := facts{sessions: 3, tokens: 1000, lines: 40, linesOK: true}
	full.lineRate.Add("claude-code", 40, 1000, 2, true)
	full.perHundred = perHundred(&full)

	mixed := full
	mixed.lineRate.Add("gemini-cli", 0, 3000, 6, true)
	mixed.perHundred = perHundred(&mixed)

	blind := facts{sessions: 3, tokens: 3000}
	blind.lineRate.Add("gemini-cli", 0, 3000, 6, true)
	blind.perHundred = perHundred(&blind)

	tests := []struct {
		name            string
		f               facts
		inHook, inPost  bool
		inLedger        bool
		noteHas, absent string
	}{
		{"covers everything", full, true, true, true, "not a bill", ""},
		{"leaves usage out", mixed, false, false, true, "75% of tokens left out", ""},
		{"no source records lines", blind, false, false, false, "", "0 lines"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.f
			if got := strings.Contains(moneySub(&f), "per 100 lines"); got != tt.inHook {
				t.Fatalf("hook %q: ratio shown = %v, want %v", moneySub(&f), got, tt.inHook)
			}
			if got := strings.Contains(fameLine(&f), "per 100"); got != tt.inPost {
				t.Fatalf("post %q: ratio shown = %v, want %v", fameLine(&f), got, tt.inPost)
			}
			var ledger string
			for _, s := range ledgerStats(&f) {
				if s.Label == "per 100 lines" {
					ledger = s.Note
				}
			}
			if (ledger != "") != tt.inLedger || !strings.Contains(ledger, tt.noteHas) {
				t.Fatalf("ledger note %q, want present=%v containing %q", ledger, tt.inLedger, tt.noteHas)
			}
			if tt.absent != "" {
				for _, line := range []string{scaleSub(&f), outputSub(&f), fameLine(&f)} {
					if strings.Contains(line, tt.absent) {
						t.Fatalf("%q prints %q for a window no source could measure", line, tt.absent)
					}
				}
			}
		})
	}
}

// TestTheCardQuotesTheWindowsLineRate: the ledger's per-100-lines figure is the one `status`
// divides over the same usage, not a figure the card derives on its own.
func TestTheCardQuotesTheWindowsLineRate(t *testing.T) {
	a, in := buildSample(t)
	inv := report.BuildInventory(in.Usage, in.Prices)
	r := inv.LineRate.Per100()
	if r == nil {
		t.Fatal("the sample window has no line rate")
	}
	for _, s := range a.Ledger {
		if s.Label == "per 100 lines" {
			if want := "$" + humanize.USDCell(*r); s.Value != want {
				t.Fatalf("per 100 lines = %q, want %q", s.Value, want)
			}
			return
		}
	}
	t.Fatalf("ledger %v carries no per-100-lines figure", a.Ledger)
}
