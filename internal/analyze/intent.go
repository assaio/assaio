package analyze

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/assaio/assaio/internal/label"
	"github.com/assaio/assaio/internal/store"
)

const (
	intentName      = "intent"
	intentTitle     = "Task Intent Coverage"
	intentDescribe  = "How much of the window carries a task label, and whether that is enough to read any other metric per kind of work."
	intentHowToRead = "Intent is the one thing session logs never contain: what the work was for. Labels come from 'assaio-agent mark' and let every other metric be filtered per kind of work, which is what makes \"did this hold up\" answerable instead of an average over research, refactors and greenfield features. This measures readiness to stratify, not diligence -- an unlabeled session is fully counted everywhere and is never a failure."
	// intentComparableClass is how many sessions a task class needs before a filtered read
	// of it means anything -- the same floor the confidence envelope applies.
	intentComparableClass = confidenceSampleFloor
	// intentComparableClasses is how many such classes it takes to compare kinds of work at
	// all: with one class there is nothing to compare it against.
	intentComparableClasses = 2
)

func init() { Register(intentValidator{}) }

// intentValidator reports label coverage and whether the labeled sessions are spread across
// enough task classes to support a filtered read.
//
// It deliberately has no unfavorable verdict. The only thing it could call "watch" is a
// person not labeling enough, or labeling work that is genuinely all of one kind -- and
// scoring either would turn optional bookkeeping into a judgement about how someone works,
// which is the line this project does not cross.
type intentValidator struct{}

func (intentValidator) Name() string     { return intentName }
func (intentValidator) Title() string    { return intentTitle }
func (intentValidator) Describe() string { return intentDescribe }

//nolint:gocritic // Input is required by the Validator interface; analyzed once per run, not a hot path.
func (intentValidator) Analyze(in Input) Result {
	r := Result{Name: intentName, Title: intentTitle, Describe: intentDescribe, HowToRead: intentHowToRead}
	if len(in.Sessions) == 0 {
		r.Read = noDataRead
		r.Takeaway = "No sessions in this window."
		return r
	}
	tally := tallyIntent(in.Sessions)
	// The observation is a session examined for a label, not a session that turned out to
	// have one: "none of your 331 sessions is labeled" is a confident answer, not an
	// under-sampled one, and resting on the labeled count would report it as insufficient.
	r.restsOn(len(in.Sessions), "sessions")
	r.Purity = clamp01(fracOf(int64(tally.labeled), int64(len(in.Sessions))))

	if tally.labeled == 0 {
		r.Read = noDataRead
		r.Figures = []Figure{{Label: "labeled sessions", Value: fmt.Sprintf("0 of %d", len(in.Sessions))}}
		r.Takeaway = "Nothing is labeled yet -- 'assaio-agent mark --task <kind>' after a session unlocks reading every metric per kind of work."
		r.Caveats = intentCaveats
		return r
	}

	// Never readFor: its unfavorable branch is "watch", and see the type's own comment for
	// why this metric has no unfavorable state.
	r.Read = Read{Key: "neutral", Label: "BUILDING"}
	if tally.comparableClasses >= intentComparableClasses {
		r.Read = Read{Key: "good", Label: "STRATIFIABLE"}
	}
	r.Figures = []Figure{
		{
			Label: "labeled sessions", Value: fmt.Sprintf("%d of %d", tally.labeled, len(in.Sessions)),
			Note: honestPercent(r.Purity),
		},
		{Label: "task classes used", Value: strconv.Itoa(len(tally.byTask))},
		{
			Label: "classes big enough to compare", Value: strconv.Itoa(tally.comparableClasses),
			Note: fmt.Sprintf("at least %d sessions each", intentComparableClass),
		},
		{Label: "outcome recorded", Value: fmt.Sprintf("%d of %d labeled", tally.withOutcome, tally.labeled)},
	}
	r.Bars = intentBars(tally.byTask)
	r.Takeaway = intentTakeaway(tally)
	r.Caveats = intentCaveats
	return r
}

// intentCaveats travel with every populated read. The resume caveat is the honest one: a
// Claude session id outlives a single sitting, so a label can end up covering work the
// session only took on later.
var intentCaveats = []string{
	"Labels are optional and local: unlabeled sessions stay fully counted in every metric, and 'sync' never sends them.",
	"A session id is reused across --resume, so one label can span work that changed character later in the same session.",
}

// intentTally is what one pass over the window's sessions yields.
type intentTally struct {
	labeled           int
	withOutcome       int
	byTask            map[string]int
	comparableClasses int
}

func tallyIntent(sessions []store.SessionRow) intentTally {
	t := intentTally{byTask: map[string]int{}}
	for i := range sessions {
		s := &sessions[i]
		if s.Task == "" && s.Outcome == "" && s.Difficulty == "" {
			continue
		}
		t.labeled++
		if s.Outcome != "" {
			t.withOutcome++
		}
		if s.Task != "" {
			t.byTask[s.Task]++
		}
	}
	for _, n := range t.byTask {
		if n >= intentComparableClass {
			t.comparableClasses++
		}
	}
	return t
}

// intentBars ranks the task classes by session count. The labels are a fixed vocabulary the
// tool defines, not names a person chose, so BarsPseudonym stays unset and an anonymized
// dashboard shows them as they are.
func intentBars(byTask map[string]int) []Bar {
	names := make([]string, 0, len(byTask))
	maxN := 0
	for name, n := range byTask {
		names = append(names, name)
		if n > maxN {
			maxN = n
		}
	}
	sort.Slice(names, func(i, j int) bool {
		if byTask[names[i]] != byTask[names[j]] {
			return byTask[names[i]] > byTask[names[j]]
		}
		return names[i] < names[j]
	})
	bars := make([]Bar, 0, len(names))
	for _, name := range names {
		bars = append(bars, Bar{
			Label: name,
			Value: strconv.Itoa(byTask[name]),
			Frac:  fracOf(int64(byTask[name]), int64(maxN)),
		})
	}
	return bars
}

func intentTakeaway(t intentTally) string {
	if t.comparableClasses >= intentComparableClasses {
		return fmt.Sprintf("%d task classes have enough sessions to compare -- run any metric with --%s <kind> to read it per kind of work.",
			t.comparableClasses, label.Task)
	}
	return fmt.Sprintf("%d sessions labeled so far; a second task class with %d or more sessions makes filtered comparisons meaningful.",
		t.labeled, intentComparableClass)
}
