package analyze

import (
	"fmt"
	"strconv"

	"github.com/assaio/assaio/internal/humanize"

	"github.com/assaio/assaio/internal/parser"
)

const (
	rhythmName     = "rhythm"
	rhythmTitle    = "Work Rhythm"
	rhythmDescribe = "When AI sessions run -- the off-hours and weekend share -- and how long the longest focused sessions last."
	// rhythmHowToRead is Result.HowToRead for this validator -- see its doc comment.
	rhythmHowToRead = "A signal about when the work happens, read across the whole window rather than attributed to an individual: a high off-hours share or frequent marathon sessions is a question about how work is scheduled, not a verdict on anyone's effort."
	// rhythmDayStart and rhythmDayEnd bound ordinary working hours; a session starting
	// outside them, or on a weekend, counts as off-hours.
	rhythmDayStart = 8
	rhythmDayEnd   = 18
	// rhythmOffHoursCeiling is the off-hours session share above which the rhythm reads
	// as strained rather than steady.
	rhythmOffHoursCeiling = 0.25
	// rhythmMarathonMinutes is the focused-work length at which a session counts as a
	// marathon -- long enough that both context and attention degrade.
	rhythmMarathonMinutes = 90
	// rhythmMarathonCeiling is the marathon share above which sessions read as running
	// too long too often.
	rhythmMarathonCeiling = 0.15
	// rhythmMinSessions is the floor below which timing shares are too thin to call: two
	// evening sessions are a coincidence, not a pattern.
	rhythmMinSessions = 5
)

func init() { Register(rhythmValidator{}) }

// rhythmValidator reads the window's working pattern: when sessions start and how long
// the longest focused stretches run.
type rhythmValidator struct{}

func (rhythmValidator) Name() string     { return rhythmName }
func (rhythmValidator) Title() string    { return rhythmTitle }
func (rhythmValidator) Describe() string { return rhythmDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (rhythmValidator) Analyze(in Input) Result {
	r := Result{Name: rhythmName, Title: rhythmTitle, Describe: rhythmDescribe, HowToRead: rhythmHowToRead}
	timed := timedSessions(in.Sessions)
	if len(timed) == 0 {
		r.noData("sessions", "No usage in this window.")
		return r
	}
	// When work starts is readable from any source that stamps a session; how long it ran is
	// not. Focused minutes are the gaps between turns, so a whole-session record has none to
	// measure and its structural zero would read as a session that finished in an instant.
	paced, pacedShare := sessionsAnswering(timed, parser.SignalActiveMinutes)
	r.restsOn(len(timed), "sessions")
	// The reach declared is the length half's, but only while there is a length half to
	// narrow: with none, the figures that survive describe the whole window, and declaring
	// zero would print "nothing in this window can answer it" above a complete off-hours
	// share. The withheld Read and the takeaway carry that half's absence instead.
	if len(paced) > 0 {
		r.covering(pacedShare)
	}
	p := buildRhythm(timed, paced)
	// Both halves need the floor, not just the window: a marathon share read off two timed
	// sessions is the same coin flip as an off-hours share read off two.
	sufficient := len(timed) >= rhythmMinSessions && len(paced) >= rhythmMinSessions
	calm := p.OffHoursShare <= rhythmOffHoursCeiling && p.MarathonShare <= rhythmMarathonCeiling
	steady := calm && sufficient

	r.Read = rhythmRead(sufficient, steady)
	r.Purity = rhythmPurity(p, sufficient)
	r.Figures = []Figure{
		{Label: "sessions timed", Value: strconv.Itoa(len(timed))},
		{Label: "off-hours", Value: humanize.PercentAt(p.OffHoursShare, 0), Note: humanize.PercentAt(p.WeekendShare, 0) + " on weekends"},
		basisFigure("longest sessions", minutesLabel(p.P95ActiveMinutes)+" p95", "focused work", len(paced)),
		basisFigure("marathons", humanize.PercentAt(p.MarathonShare, 0), "over "+strconv.Itoa(rhythmMarathonMinutes)+" min focused", len(paced)),
	}
	r.Bars = rhythmBands(p.Bands)
	r.Takeaway = rhythmTakeaway(steady, sufficient, len(paced))
	r.Caveats = append(
		r.Caveats,
		"Prov.: hours are read in this machine's local timezone; sessions recorded in another zone land in the wrong band.",
		"Aggregate workload signal -- not an input to evaluating an individual.",
	)
	if len(paced) < len(timed) {
		r.Caveats = append(r.Caveats, rhythmCoverageCaveat(len(paced), len(timed)))
	}
	return r
}

// rhythmCoverageCaveat states how many sessions the length figures rest on, so the ones a
// source could not time read as absent rather than as instant.
func rhythmCoverageCaveat(paced, timed int) string {
	return fmt.Sprintf(
		"Prov.: session length reads the %d of %d sessions whose source records focused minutes; the rest are absent from it, not zero-length.",
		paced, timed,
	)
}

// rhythmRead reports the neutral no-verdict Read while too few timed sessions exist to
// call a pattern, rather than the alarming WATCH a plain readFor would hand a window whose
// every measured share is perfect.
func rhythmRead(sufficient, steady bool) Read {
	if !sufficient {
		return noDataRead
	}
	return readFor(steady, "Steady")
}

// rhythmPurity is high when work sits inside ordinary hours and sessions stay a sane
// length; too small a sample yields the neutral 0.5 the other validators use for
// "not enough evidence yet".
func rhythmPurity(p rhythmProfile, sufficient bool) float64 {
	if !sufficient {
		return 0.5
	}
	return clamp01(1 - (p.OffHoursShare+p.MarathonShare)/2)
}

// rhythmTakeaway makes no workload judgement below the session floor: it prints verbatim
// beside a Read that withholds its verdict there.
func rhythmTakeaway(steady, sufficient bool, paced int) string {
	switch {
	case paced == 0:
		return "No source in this window records focused minutes, so how long sessions run cannot be read -- the timing shares above are shown without a verdict."
	case !sufficient:
		return "Too few sessions this window to call the rhythm -- the shares above are shown without a verdict."
	case steady:
		return "Sessions sit inside ordinary hours and stay a reasonable length."
	default:
		return "A large share of sessions runs off-hours or very long -- worth looking at how the work is scheduled."
	}
}
