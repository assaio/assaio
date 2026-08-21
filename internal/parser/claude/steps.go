package claude

import (
	"strconv"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// stepRecorder builds a transcript's step sequence beside its usage records: what the agent
// did, in what order, and how each step ended. It is driven by parseState rather than by a
// second scan, so exactly one order of checks runs over a line and the two readings cannot
// disagree about what that line was.
//
// targets maps a file path to the small integer that stands for it, in memory for the lifetime
// of one parse only -- the same discipline reworkTracker follows, and for the same reason
// (PRIVACY.md promises no path is persisted). What reaches the store is the integer.
type stepRecorder struct {
	steps   []usage.Step
	targets map[string]int64
	// byToolUse indexes a tool call's step by the id its result will quote, so an error, a
	// denial or an edit target lands on the call it belongs to rather than on the latest one.
	byToolUse map[string]int
	// byResponse indexes one API response's step by the key the records dedupe on, because
	// Claude writes a response as several lines that each repeat its usage.
	byResponse map[string]int
	// callsPerResponse counts tool calls already keyed under one response, for the transcript
	// that carries no call ids. Counting per response rather than per line is the point:
	// keying on the line's block index collapsed three real calls into one row.
	callsPerResponse map[string]int
	ordinal          int64
}

func newStepRecorder() *stepRecorder {
	return &stepRecorder{
		targets:          map[string]int64{},
		byToolUse:        map[string]int{},
		byResponse:       map[string]int{},
		callsPerResponse: map[string]int{},
	}
}

// assistant records one model response. tokens comes from the record this response folded
// into, so the two readings cannot reach different totals by different arithmetic: the record
// keeps a per-field maximum across a response's lines, and a maximum of per-line sums stops
// being the same number as soon as one field is not monotone.
func (s *stepRecorder) assistant(l *line, key string, tokens int64) {
	if at, ok := s.byResponse[key]; ok {
		s.steps[at].Tokens = tokens
		if s.steps[at].Outcome == "" {
			s.steps[at].Outcome = stopOutcome(l.Message.StopReason)
		}
		return
	}
	s.append(&usage.Step{
		DedupeKey: key,
		Timestamp: l.Timestamp,
		Kind:      usage.StepAssistant,
		Outcome:   stopOutcome(l.Message.StopReason),
		Model:     l.Message.Model,
		Tokens:    tokens,
	})
	s.byResponse[key] = len(s.steps) - 1
}

// toolCalls records one step per tool_use block, in the order the response emitted them. The
// key is the call's own id, which is what a tool result quotes and is unique across the
// transcript. cwd is the session's working directory, which resolves a target named relatively.
func (s *stepRecorder) toolCalls(l *line, key, cwd string, blocks []contentBlock) {
	for _, b := range blocks {
		if b.Type != "tool_use" {
			continue
		}
		dedupe := b.ID
		if dedupe == "" {
			dedupe = key + ":" + strconv.Itoa(s.callsPerResponse[key])
			s.callsPerResponse[key]++
		}
		s.append(&usage.Step{
			DedupeKey: dedupe,
			Timestamp: l.Timestamp,
			Kind:      parser.StepKind(b.Name),
			Model:     l.Message.Model,
			TargetRef: s.ref(b.targetPath(), cwd),
		})
		if b.ID != "" {
			s.byToolUse[b.ID] = len(s.steps) - 1
		}
	}
}

// resolveResults settles the outcome of every call a line answers. A result quoting an id this
// recorder never saw is dropped rather than attributed to the nearest step: a wrong
// attribution is worse than a missing one.
func (s *stepRecorder) resolveResults(blocks []contentBlock) {
	for _, b := range blocks {
		if b.Type != "tool_result" {
			continue
		}
		i, ok := s.byToolUse[b.ToolUseID]
		if !ok {
			continue
		}
		if b.IsError {
			s.steps[i].Outcome = usage.OutcomeError
			continue
		}
		s.steps[i].Outcome = usage.OutcomeOK
	}
}

// denial marks the declined call, which the denial line names in its own tool_result block.
// Reading that id is what makes this correct: a backward walk to the most recent unsettled
// step put 42 of 497 real denials on a different call, and 30 of those were then overwritten
// by that call's own result, so the denial left the timeline entirely.
func (s *stepRecorder) denial(blocks []contentBlock) {
	for _, b := range blocks {
		if b.Type != "tool_result" || b.ToolUseID == "" {
			continue
		}
		if i, ok := s.byToolUse[b.ToolUseID]; ok {
			s.steps[i].Outcome = usage.OutcomeDenied
			return
		}
	}
}

// compaction records the context overflow itself as a step, so a sequence shows where the
// agent lost its context rather than only how often that happened. The caller collapses the
// pair of adjacent markers Claude Code writes for one overflow, using the same flag the record
// path uses, so neither can count it differently.
func (s *stepRecorder) compaction(l *line) {
	if l.UUID == "" {
		return
	}
	s.append(&usage.Step{
		DedupeKey: l.UUID + ":compaction",
		Timestamp: l.Timestamp,
		Kind:      usage.StepCompaction,
	})
}

// ref is the integer standing for path, assigned in first-seen order within one parse, and 0 for a
// call that names no file. A relative path is resolved against the session's cwd: the same file
// named both ways would otherwise hold two refs and split the repeat count a detector reads -- 29
// of 3,516 read calls in a 400-transcript sample name one relatively. A relative path with no cwd
// to resolve it against has no honest identity and gets none.
func (s *stepRecorder) ref(path, cwd string) int64 {
	key := targetKey(path, cwd)
	if key == "" {
		return 0
	}
	if ref, seen := s.targets[key]; seen {
		return ref
	}
	ref := int64(len(s.targets)) + 1
	s.targets[key] = ref
	return ref
}

func (s *stepRecorder) append(st *usage.Step) {
	s.ordinal++
	st.Ordinal = s.ordinal
	st.Tool = tool
	s.steps = append(s.steps, *st)
}

// stamp fills in the session and the timeline every step belongs to. It runs once at the end
// because a transcript's first lines can precede the line that names either.
func (s *stepRecorder) stamp(sessionID, timeline string) []usage.Step {
	for i := range s.steps {
		s.steps[i].SessionID = sessionID
		s.steps[i].Timeline = timeline
	}
	return s.steps
}

// stopOutcome maps Claude's stop_reason onto the closed outcome vocabulary. "tool_use" is a
// response that continued into a call rather than one that ended, and is left unstated: the
// calls it made carry their own outcomes, and reading it as OutcomeOK would count one turn's
// success twice.
func stopOutcome(stop string) string {
	switch stop {
	case "end_turn", "stop_sequence":
		return usage.OutcomeOK
	case "max_tokens":
		return usage.OutcomeTruncated
	default:
		return ""
	}
}
