package signal

import (
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/parser"
)

// A catalog entry that omits a field is worse than no entry: it looks authoritative and says
// nothing. Every declared signal must answer every question the catalog exists to answer.
func TestEverySignalIsFullyDeclared(t *testing.T) {
	statuses := map[string]bool{Observed: true, Estimated: true, Derived: true}
	grains := map[string]bool{GrainStep: true, GrainTurn: true, GrainSession: true, GrainDay: true, GrainWindow: true}

	seen := map[string]bool{}
	for _, s := range Catalog() {
		if seen[s.ID] {
			t.Errorf("duplicate signal id %s", s.ID)
		}
		seen[s.ID] = true
		if strings.Count(s.ID, ".") < 2 {
			t.Errorf("%s must be <domain>.<thing>.<measure>", s.ID)
		}
		for field, v := range map[string]string{
			"Title": s.Title, "Unit": s.Unit, "ZeroMeans": s.ZeroMeans, "Describe": s.Describe,
		} {
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s has no %s", s.ID, field)
			}
		}
		if !statuses[s.Status] {
			t.Errorf("%s has status %q outside the vocabulary", s.ID, s.Status)
		}
		if len(s.Grains) == 0 {
			t.Errorf("%s declares no grain it is honest at", s.ID)
		}
		for _, g := range s.Grains {
			if !grains[g] {
				t.Errorf("%s claims unknown grain %q", s.ID, g)
			}
		}
	}
}

// ADR 0008 names "attributed" but defers it to B85, which will produce the first signal
// deserving it. Registering it early would let a signal claim a status nothing can compute.
func TestAttributedIsNotRegisteredYet(t *testing.T) {
	for _, s := range Catalog() {
		if s.Status == "attributed" {
			t.Fatalf("%s claims attributed status before attribution edges exist", s.ID)
		}
	}
}

func TestGetFindsAndRejects(t *testing.T) {
	if _, ok := Get("ai.tokens.total"); !ok {
		t.Fatal("ai.tokens.total must be in the catalog")
	}
	if _, ok := Get("ai.vibes.total"); ok {
		t.Fatal("an unknown id must not resolve")
	}
}

func TestCatalogIsSortedAndCopied(t *testing.T) {
	got := Catalog()
	for i := 1; i < len(got); i++ {
		if got[i-1].ID > got[i].ID {
			t.Fatalf("catalog is not sorted: %s before %s", got[i-1].ID, got[i].ID)
		}
	}
	got[0].ID = "mutated"
	if Catalog()[0].ID == "mutated" {
		t.Fatal("Catalog exposes its backing array")
	}
}

// Capability is declared by the source, meaning by the catalog. A source claiming a signal
// nobody declared is the drift that would make coverage report support for nothing.
func TestEverySourceAnswersOnlyDeclaredSignals(t *testing.T) {
	declared := map[string]bool{}
	for _, s := range Catalog() {
		declared[s.ID] = true
	}
	for _, d := range depthRows() {
		for _, id := range d.Answers {
			if !declared[id] {
				t.Errorf("%s claims to answer %s, which no signal declares", d.Tool, id)
			}
		}
	}
}

// The three depth axes summarise; Answers is the per-signal truth. They must not contradict:
// a source claiming activity has to answer at least one activity signal, and one claiming
// none must answer no such signal.
func TestDepthAxesAgreeWithAnswers(t *testing.T) {
	activity := []string{"ai.lines.added", "ai.lines.removed", "ai.edits.count", "ai.tool_calls.count"}
	attribution := []string{"ai.skill.tokens", "ai.agent.tokens"}
	for _, d := range depthRows() {
		if got := answersAny(d.Answers, activity); got != d.Activity {
			t.Errorf("%s declares Activity=%v but answers activity signals=%v", d.Tool, d.Activity, got)
		}
		if got := answersAny(d.Answers, attribution); got != d.Attribution {
			t.Errorf("%s declares Attribution=%v but answers attribution signals=%v", d.Tool, d.Attribution, got)
		}
	}
}

func answersAny(have, want []string) bool {
	for _, w := range want {
		for _, h := range have {
			if h == w {
				return true
			}
		}
	}
	return false
}

// depthRows reads the matrix one row at a time through the two questions the package
// publishes, so the contract tests need no bulk accessor nothing in the binary calls.
func depthRows() []parser.Depth {
	tools := parser.Tools()
	out := make([]parser.Depth, 0, len(tools))
	for _, tool := range tools {
		if d, ok := parser.DepthOf(tool); ok {
			out = append(out, d)
		}
	}
	return out
}
