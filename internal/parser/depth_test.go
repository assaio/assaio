package parser

import "testing"

func TestDepthTiersMatchWhatEachParserExtracts(t *testing.T) {
	tests := []struct {
		tool                          string
		tier                          string
		tokens, activity, attribution bool
	}{
		{"claude-code", Deep, true, true, true},
		{"codex", Standard, true, true, false},
		{"gemini-cli", Standard, true, false, false},
		{"copilot-cli", Standard, true, true, false},
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
	for _, d := range depths {
		if d.Tier != Deep && len(d.Gaps) == 0 {
			t.Errorf("%s is %q but documents no gaps", d.Tool, d.Tier)
		}
	}
}

func TestDepthTiersComeFromTheFixedVocabulary(t *testing.T) {
	for _, d := range depths {
		switch d.Tier {
		case Deep, Standard, ImportOnly:
		default:
			t.Errorf("%s has unknown tier %q", d.Tool, d.Tier)
		}
	}
}

// Tools is what every surface naming a source reads, so a parser that lands without a row
// here would be silently un-syncable and un-clearable -- which is how copilot-cli shipped.
func TestToolsNamesEveryInTreeSource(t *testing.T) {
	got := Tools()
	if len(got) != len(depths) {
		t.Fatalf("Tools() = %v, want one name per depth row", got)
	}
	for i, d := range depths {
		if got[i] != d.Tool {
			t.Errorf("Tools()[%d] = %q, want %q", i, got[i], d.Tool)
		}
	}
}

// Answers is the single capability question every surface asks. A source outside the matrix
// is an exec plugin, whose protocol carries tokens and nothing else.
func TestAnswersReadsTheMatrixAndThePluginFloor(t *testing.T) {
	tests := []struct {
		tool, signal string
		want         bool
	}{
		{"claude-code", "ai.lines.added", true},
		{"claude-code", "ai.tool_errors.count", true},
		{"codex", "ai.tool_errors.count", false},
		{"gemini-cli", "ai.lines.added", false},
		{"copilot-cli", "ai.lines.added", true},
		{"copilot-cli", "ai.edits.count", false},
		{"copilot-cli", "ai.tokens.reasoning", true},
		{"plugin:acme", "ai.tokens.total", true},
		{"plugin:acme", "ai.edits.count", false},
		// A name that is neither a matrix row nor a plugin is not a usage source at all --
		// the git collector is one, and it must not inherit the exec protocol's token floor.
		{"git", "ai.tokens.total", false},
		{"", "ai.tokens.total", false},
	}
	for _, tt := range tests {
		if got := Answers(tt.tool, tt.signal); got != tt.want {
			t.Errorf("Answers(%q, %q) = %v, want %v", tt.tool, tt.signal, got, tt.want)
		}
	}
}

// The exec parser protocol lets a plugin declare turn or session grain per record, so no
// plugin can be assumed to answer a signal that needs a second timestamp -- the same rule
// that keeps copilot-cli out of the turn count.
func TestAPluginAnswersNoPerTurnSignal(t *testing.T) {
	for _, id := range []string{"ai.turns.count", "ai.session.active_minutes"} {
		if Answers("plugin:acme", id) {
			t.Errorf("a plugin cannot be assumed to answer %s: its grain is per record", id)
		}
	}
}

func TestDepthsAreOrderedDeepestFirst(t *testing.T) {
	got := depths
	if len(got) == 0 {
		t.Fatal("no depths")
	}
	if got[0].Tier != Deep {
		t.Errorf("first entry = %q, want the deepest source first", got[0].Tier)
	}
}

// The tier table's Activity bit and full activity capture are different questions, and
// Copilot CLI is the source that makes them differ. They must never be read as one.
func TestFullActivityIsStricterThanTheActivityAxis(t *testing.T) {
	d, ok := DepthOf("copilot-cli")
	if !ok {
		t.Fatal("copilot-cli missing from the matrix")
	}
	if !d.Activity {
		t.Fatal("copilot-cli records changed lines, so the tier axis is true")
	}
	if HasFullActivity("copilot-cli") {
		t.Error("it answers no edit, tool-call or rework signal, so activity capture is not full")
	}
	if !HasLineOutput("copilot-cli") {
		t.Error("it does contribute changed lines")
	}
	for _, tool := range []string{"claude-code", "codex"} {
		if !HasFullActivity(tool) {
			t.Errorf("%s answers every activity signal", tool)
		}
	}
	for _, tool := range []string{"gemini-cli", "cline", "plugin:acme", "git"} {
		if HasFullActivity(tool) || HasLineOutput(tool) {
			t.Errorf("%s contributes no line or edit signals", tool)
		}
	}
}
