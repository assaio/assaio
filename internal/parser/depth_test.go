package parser

import "testing"

// TestEveryInTreeSourceDeclaresItsDepth keeps the matrix from silently going stale: a new
// parser that never appears here would read as "no depth known" everywhere it is surfaced.
func TestEveryInTreeSourceDeclaresItsDepth(t *testing.T) {
	for _, tool := range []string{"claude-code", "codex", "gemini-cli", "cline"} {
		if _, ok := DepthOf(tool); !ok {
			t.Errorf("%s has no depth entry", tool)
		}
	}
}

func TestDepthTiersMatchWhatEachParserExtracts(t *testing.T) {
	tests := []struct {
		tool                          string
		tier                          string
		tokens, activity, attribution bool
	}{
		{"claude-code", Deep, true, true, true},
		{"codex", Standard, true, true, false},
		{"gemini-cli", Standard, true, false, false},
		{"cline", Standard, true, false, false},
	}
	for _, tt := range tests {
		got, ok := DepthOf(tt.tool)
		if !ok {
			t.Fatalf("%s missing", tt.tool)
		}
		if got.Tier != tt.tier || got.Tokens != tt.tokens ||
			got.Activity != tt.activity || got.Attribution != tt.attribution {
			t.Errorf("%s = %+v, want tier=%s tokens=%v activity=%v attribution=%v",
				tt.tool, got, tt.tier, tt.tokens, tt.activity, tt.attribution)
		}
	}
}

// TestAnythingLessThanDeepDocumentsItsGaps is the honesty rule the tiers rest on: calling a
// source "standard" is only fair if a reader can see exactly what it does not carry.
func TestAnythingLessThanDeepDocumentsItsGaps(t *testing.T) {
	for _, d := range Depths() {
		if d.Tier != Deep && len(d.Gaps) == 0 {
			t.Errorf("%s is %q but documents no gaps", d.Tool, d.Tier)
		}
	}
}

func TestDepthTiersComeFromTheFixedVocabulary(t *testing.T) {
	for _, d := range Depths() {
		switch d.Tier {
		case Deep, Standard, ImportOnly:
		default:
			t.Errorf("%s has unknown tier %q", d.Tool, d.Tier)
		}
	}
}

// TestHasActivityIsFalseForSourcesOutsideTheMatrix keeps an exec plugin -- whose records
// carry no line or edit fields at all -- from being counted as activity-capable.
func TestHasActivityIsFalseForSourcesOutsideTheMatrix(t *testing.T) {
	for _, tool := range []string{"plugin:acme", "", "unknown-tool"} {
		if HasActivity(tool) {
			t.Errorf("HasActivity(%q) = true, want false", tool)
		}
	}
	if !HasActivity("claude-code") || !HasActivity("codex") {
		t.Error("the two activity-extracting parsers must report true")
	}
	if HasActivity("gemini-cli") || HasActivity("cline") {
		t.Error("token-only parsers must not report activity")
	}
}

func TestDepthsAreOrderedDeepestFirst(t *testing.T) {
	got := Depths()
	if len(got) == 0 {
		t.Fatal("no depths")
	}
	if got[0].Tier != Deep {
		t.Errorf("first entry = %q, want the deepest source first", got[0].Tier)
	}
}
