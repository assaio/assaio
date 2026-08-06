package analyze

import (
	"sort"
	"strconv"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/parser"
)

const (
	turnEffName      = "turn-efficiency"
	turnEffTitle     = "Turn Efficiency"
	turnEffDescribe  = "Getting more per prompt: one-shot rate, median turns per code-producing session, and output tokens per turn."
	turnEffHowToRead = "These are prompting-efficiency signals, not quality. A session that lands an edit in a turn or two is efficient; a long session can be deliberate, careful work. Task size is invisible here, so read it directionally, never as a per-person score."
	// turnEffOneShotMax is the turn count at or below which a code-producing session is one-shot.
	turnEffOneShotMax = 2
	// turnEffGoodOneShot is the one-shot rate above which prompting reads as efficient.
	turnEffGoodOneShot = 0.25
	// turnEffMinCodeSessions is the floor below which the one-shot rate is too thin to judge.
	turnEffMinCodeSessions = 5
)

func init() { Register(turnEffValidator{}) }

// turnEffValidator measures prompting efficiency over code-producing sessions: how often a
// result lands in one or two turns, the median turns it takes, and output produced per turn.
type turnEffValidator struct{}

func (turnEffValidator) Name() string     { return turnEffName }
func (turnEffValidator) Title() string    { return turnEffTitle }
func (turnEffValidator) Describe() string { return turnEffDescribe }

//nolint:gocritic // Input is required by the Validator interface; analyzed once per run, not a hot path.
func (turnEffValidator) Analyze(in Input) Result {
	r := Result{Name: turnEffName, Title: turnEffTitle, Describe: turnEffDescribe, HowToRead: turnEffHowToRead}
	// A session counts here only if its source records both halves of the question: edits,
	// or a zero edit count is the tool's silence and skipping the session as "not
	// code-producing" judges prompting from a field the tool never kept; and turns, since
	// every figure below divides by them and a source that totals a session would make each
	// one read as landing in no turns at all.
	edited, coverage := sessionsAnswering(in.Sessions, parser.SignalEditsCount, parser.SignalTurnsCount)
	r.covering(coverage)
	var codeSessions, oneShot int64
	var codeTurns, outPerTurn []float64
	for i := range edited {
		s := &edited[i]
		if s.Edits == 0 {
			continue
		}
		codeSessions++
		codeTurns = append(codeTurns, float64(s.Turns))
		if s.Turns <= turnEffOneShotMax {
			oneShot++
		}
		if s.Turns > 0 {
			outPerTurn = append(outPerTurn, float64(s.OutputTokens)/float64(s.Turns))
		}
	}
	if codeSessions == 0 {
		r.noData("code-producing sessions", turnEffEmptyTakeaway(len(edited)))
		return r
	}
	r.restsOn(int(codeSessions), "code-producing sessions")
	enough := codeSessions >= turnEffMinCodeSessions
	oneShotRate := fracOf(oneShot, codeSessions)
	sort.Float64s(codeTurns)

	if enough {
		r.Read = readFor(oneShotRate >= turnEffGoodOneShot, "Efficient")
	} else {
		r.Read = noDataRead
	}
	r.Purity = clamp01(oneShotRate)
	r.Figures = []Figure{
		{Label: "one-shot rate", Value: humanize.Percent(oneShotRate), Note: "code sessions in <=" + strconv.Itoa(turnEffOneShotMax) + " turns"},
		{Label: "median turns", Value: strconv.FormatFloat(medianAt50(codeTurns), 'f', 0, 64), Note: "per code session"},
		{Label: "output/turn", Value: medianOutputPerTurn(outPerTurn), Note: "median tokens"},
	}
	r.Takeaway = turnEffTakeaway(enough, oneShotRate)
	r.Caveats = []string{"Task size is invisible in logs, so a low one-shot rate can mean hard problems, not weak prompting -- directional only."}
	return r
}

// turnEffEmptyTakeaway separates a window whose sessions edited nothing from one whose
// sources never record an edit -- the second says nothing about how anyone prompts.
func turnEffEmptyTakeaway(edited int) string {
	if edited == 0 {
		return "No source in this window records file edits, so prompting efficiency cannot be read from it."
	}
	return "No code-producing sessions in this window."
}

// medianOutputPerTurn renders the median output-tokens-per-turn, or "—" when no session had turns.
func medianOutputPerTurn(outPerTurn []float64) string {
	if len(outPerTurn) == 0 {
		return "—"
	}
	sort.Float64s(outPerTurn)
	return humanize.Count(int64(medianAt50(outPerTurn)))
}

func turnEffTakeaway(enough bool, oneShotRate float64) string {
	switch {
	case !enough:
		return "Too few code-producing sessions this window to judge prompting efficiency."
	case oneShotRate >= turnEffGoodOneShot:
		return "A healthy share of code sessions land in one or two turns."
	default:
		return "Most code sessions take several turns -- could be hard problems, or room to prompt more precisely."
	}
}
