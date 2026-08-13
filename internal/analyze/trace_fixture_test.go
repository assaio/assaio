package analyze

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/trace"
	"github.com/assaio/assaio/internal/usage"
)

// A sequence spelled as text, so a detector's fixture reads as the shape it is testing rather
// than as forty lines of struct literal. One field per step:
//
//	a        an assistant turn, 1,000 tokens
//	a:2500   an assistant turn of 2,500 tokens
//	e3       an edit of file 3      c    a command
//	r3       a read of file 3       s    a search
//	k        a compaction           o    some other call
//	e3!err   the same edit, failed (!den denied, !trunc truncated)
//
// Ordinals are assigned in the order written, which is the only order a sequence has.
func stepsFrom(t *testing.T, spec string) []store.TimelineStep {
	t.Helper()
	at := time.Date(2026, 8, 2, 9, 0, 0, 0, time.UTC)
	var out []store.TimelineStep
	for i, field := range strings.Fields(spec) {
		body, outcome := splitOutcome(t, field)
		kind, target, tokens := splitStep(t, body)
		out = append(out, store.TimelineStep{
			Ordinal: int64(i + 1), Timestamp: at.Add(time.Duration(i) * time.Minute),
			Kind: kind, Outcome: outcome, Model: "claude-opus-4-5", Tokens: tokens, TargetRef: target,
		})
	}
	return out
}

func splitOutcome(t *testing.T, field string) (body, outcome string) {
	body, suffix, marked := strings.Cut(field, "!")
	if !marked {
		return body, ""
	}
	switch suffix {
	case "err":
		return body, usage.OutcomeError
	case "den":
		return body, usage.OutcomeDenied
	case "trunc":
		return body, usage.OutcomeTruncated
	case "ok":
		return body, usage.OutcomeOK
	}
	t.Fatalf("step %q: unknown outcome %q", field, suffix)
	return "", ""
}

func splitStep(t *testing.T, body string) (kind string, target, tokens int64) {
	head, rest := body[:1], body[1:]
	switch head {
	case "a":
		tokens = 1000
		if amount, ok := strings.CutPrefix(rest, ":"); ok {
			tokens = mustInt(t, amount)
		}
		return usage.StepAssistant, 0, tokens
	case "e", "r":
		kind = usage.StepEdit
		if head == "r" {
			kind = usage.StepRead
		}
		return kind, mustInt(t, rest), 0
	case "c":
		return usage.StepCommand, 0, 0
	case "s":
		return usage.StepSearch, 0, 0
	case "o":
		return usage.StepOther, 0, 0
	case "k":
		return usage.StepCompaction, 0, 0
	}
	t.Fatalf("step %q: unknown kind %q", body, head)
	return "", 0, 0
}

func mustInt(t *testing.T, s string) int64 {
	t.Helper()
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		t.Fatalf("step target/tokens %q: %v", s, err)
	}
	return n
}

// sequence builds one interactive sequence from a spec. project is what the drill and the bars
// group by; "" leaves it out of both.
func sequence(t *testing.T, id, project, spec string) store.Timeline {
	t.Helper()
	return store.Timeline{
		Tool: "claude-code", SessionID: id, Entrypoint: "cli", Project: project,
		Steps: stepsFrom(t, spec),
	}
}

// subAgentSequence is the same, under a sub-agent's own timeline inside session id.
func subAgentSequence(t *testing.T, id, agent, project, spec string) store.Timeline {
	t.Helper()
	seq := sequence(t, id, project, spec)
	seq.Timeline = agent
	return seq
}

func traceOf(sequences ...store.Timeline) trace.Set { return trace.New(sequences) }

// interactiveView is the scope every detector in this package declares, with an address, since the
// readers take one: a View grew past the size the linter passes by value once it started carrying
// its own horizon.
func interactiveView(set *trace.Set) *trace.View {
	v := set.Scope(trace.Interactive)
	return &v
}
