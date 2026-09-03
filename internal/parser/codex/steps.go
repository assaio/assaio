package codex

import (
	"fmt"
	"sort"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// stepRecorder builds a rollout's step sequence beside its usage records: what the agent did,
// in what order, and what it acted on. Like Claude's, it is driven by parseState rather than by
// a second scan, so one order of checks runs over a line and the two readings cannot disagree
// about what that line was.
//
// targets maps a file path to the small integer that stands for it, in memory for the lifetime
// of one parse only -- the same discipline reworkTracker follows, and for the same reason
// (PRIVACY.md promises no path is persisted). What reaches the store is the integer.
type stepRecorder struct {
	steps   []usage.Step
	targets map[string]int64
	// turnAt indexes the assistant step opened for the turn now in progress, or -1 when no
	// response item has been seen since the last token_count. The step is opened at the turn's
	// first response item so it precedes the calls that turn made, and its tokens arrive later
	// from the record the closing token_count produces -- the same deferral pending activity
	// already uses, for the same reason: Codex prices a turn, never a message.
	turnAt int
	// patches counts patch_apply_end events seen, so each one's file steps get a dedupe key
	// that is stable across a re-read of a file that has only grown -- the assumption the
	// record key already makes with st.turn.
	patches int
	// dropped holds the provisional turns closeTurn withdrew. Held as indices rather than
	// removed in place because a turn's own tool calls are already recorded behind it by the
	// time its token_count says whether it produced anything.
	dropped map[int]bool
	// namedWrite records that this turn already recorded an edit step from a call Codex named,
	// so the patch event that follows does not record the same application a second time. The
	// record path makes the identical choice one field over (see activity.flushInto): the two
	// readings of one scan must not disagree about how many edits happened. Reset per turn,
	// because a later turn's patch is a different application.
	namedWrite bool
	// compactions counts context overflows, so each one's dedupe key is a counter rather than a
	// position in a slice a later change could compact. Same reason patches is one.
	compactions int
	// callIDs is the call ids this rollout has already keyed a step under.
	callIDs map[string]bool
}

func newStepRecorder() *stepRecorder {
	return &stepRecorder{targets: map[string]int64{}, turnAt: -1, dropped: map[int]bool{}, callIDs: map[string]bool{}}
}

// openTurn records the model response a turn begins with, once per turn. Codex writes no usage
// on the response itself, so the step is provisional: closeTurn either prices it or drops it.
func (s *stepRecorder) openTurn(st *parseState) {
	if s.turnAt >= 0 {
		return
	}
	s.steps = append(s.steps, usage.Step{
		DedupeKey: fmt.Sprintf("%s:turn:%d", st.fileFP, st.turn),
		Timestamp: st.ts,
		Kind:      usage.StepAssistant,
		Model:     st.model,
	})
	s.turnAt = len(s.steps) - 1
}

// closeTurn attaches the turn's own token total to the response that opened it. A turn whose
// token_count carried no delta produced no record and gets no step: an assistant step reading
// zero tokens would be counted by every rate over turns and priced by none of them, which is
// the dilution the capability gate exists to prevent.
func (s *stepRecorder) closeTurn(rec *usage.Record, produced bool) {
	if s.turnAt < 0 {
		return
	}
	// A later turn's patch is a different application, so the suppression namedWrite carries
	// ends with the turn that set it.
	s.namedWrite = false
	if !produced {
		s.dropped[s.turnAt] = true
		s.turnAt = -1
		return
	}
	// Summed with plain +, three fields a log can each state as MaxInt64 overflow into a negative
	// total -- which the store would take as a turn that cost less than nothing. The record's own
	// clamps hold each field; only their sum needs this one.
	s.steps[s.turnAt].Tokens = parser.SumNonNeg(rec.InputTokens, rec.CacheReadTokens, rec.OutputTokens)
	s.turnAt = -1
}

// toolCall records one call in the order it was made. Outcome is deliberately left unstated on a
// call Codex marks "completed": that word says the call returned, not that what it ran worked --
// 57 of the 840 commands in the audited corpus exited non-zero under it. Codex states the exit
// status in a second event stream that carries no call id, so there is nothing to join it to.
// Only an explicitly failed call is an error, which is the same line the record's tool-error
// count already holds.
func (s *stepRecorder) toolCall(st *parseState, callID, name, status string) {
	// The call's own id is the strongest key there is -- it is what the call's output quotes, and
	// it is stable across a re-read where a positional counter is only stable while the file
	// ahead of it does not change. It is the vendor's to keep unique, though, and the store's
	// INSERT OR IGNORE would drop the second of two steps sharing one key rather than error, so
	// a repeat falls back to a position instead of silently collapsing two calls into one row.
	key := callID
	if key == "" || s.callIDs[key] {
		key = fmt.Sprintf("call:%d", len(s.steps))
	} else {
		s.callIDs[callID] = true
	}
	step := usage.Step{
		DedupeKey: st.fileFP + ":" + key,
		Timestamp: st.ts,
		Kind:      parser.StepKind(name),
		Model:     st.model,
	}
	if failedStatus(status) {
		step.Outcome = usage.OutcomeError
	}
	if step.Kind == usage.StepEdit {
		s.namedWrite = true
	}
	s.steps = append(s.steps, step)
}

// edits records one step per file a patch touched, sorted by path so a map's iteration order
// never reaches an ordinal. Per file rather than per event on purpose: a Codex patch applies to
// 1.9 files on the audited corpus, and a sequence that collapsed them would hold one target
// where the work touched several -- which is the whole question a repeat-edit detector asks. The
// record's write count stays per call; these are two answers to two questions, not one figure
// twice.
func (s *stepRecorder) edits(st *parseState, paths []string, ok bool) {
	s.patches++
	// A build that names its patch calls already put this application in the sequence, and
	// Codex emits the call before the event that reports it applied. Recording both would show
	// one patch as an untargeted edit followed by its own files -- and a repeat-edit reading
	// would then file the untargeted one under "a call whose file could not be read", which is
	// not what it is.
	if s.namedWrite {
		return
	}
	sort.Strings(paths)
	outcome := usage.OutcomeError
	if ok {
		outcome = usage.OutcomeOK
	}
	for i, path := range paths {
		s.steps = append(s.steps, usage.Step{
			DedupeKey: fmt.Sprintf("%s:patch:%d:%d", st.fileFP, s.patches, i),
			Timestamp: st.ts,
			Kind:      usage.StepEdit,
			Outcome:   outcome,
			Model:     st.model,
			TargetRef: s.ref(path),
		})
	}
}

// compaction records the context overflow itself as a step, so a sequence shows where the agent
// lost its context rather than only how often that happened.
func (s *stepRecorder) compaction(st *parseState) {
	s.compactions++
	s.steps = append(s.steps, usage.Step{
		DedupeKey: fmt.Sprintf("%s:compaction:%d", st.fileFP, s.compactions),
		Timestamp: st.ts,
		Kind:      usage.StepCompaction,
		Model:     st.model,
	})
}

// ref is the integer standing for path, assigned in first-seen order within one parse, and 0 for
// a step that names no file.
func (s *stepRecorder) ref(path string) int64 {
	if path == "" {
		return 0
	}
	if ref, seen := s.targets[path]; seen {
		return ref
	}
	ref := int64(len(s.targets)) + 1
	s.targets[path] = ref
	return ref
}

// stamp fills in the session every step belongs to and numbers what survived, dropping the turns
// that produced no record. Ordinals are assigned here rather than at append so a dropped turn
// leaves no gap in the sequence a reader would have to explain. Codex writes no sub-agent
// transcript, so every step belongs to the one timeline a session has.
//
// A turn still open at the end of the rollout is dropped with the rest: it produced no record
// either, and the invariant worth having is that the sequence holds exactly one assistant step
// per priced turn -- 29 rollouts in the audited corpus end mid-turn, and keeping their opening
// response would have put a turn in the timeline that the usage table does not have.
func (s *stepRecorder) stamp(sessionID string) []usage.Step {
	if s.turnAt >= 0 {
		s.dropped[s.turnAt] = true
	}
	out := make([]usage.Step, 0, len(s.steps))
	var ordinal int64
	for i := range s.steps {
		if s.dropped[i] {
			continue
		}
		ordinal++
		s.steps[i].Ordinal = ordinal
		s.steps[i].Tool = tool
		s.steps[i].SessionID = sessionID
		out = append(out, s.steps[i])
	}
	return out
}
