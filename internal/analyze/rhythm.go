package analyze

import (
	"fmt"
	"strconv"

	"github.com/assaio/assaio/internal/layer"

	"github.com/assaio/assaio/internal/humanize"

	"github.com/assaio/assaio/internal/parser"
)

const (
	rhythmName     = "rhythm"
	rhythmTitle    = "Work Rhythm"
	rhythmDescribe = "When AI sessions run -- the off-hours and weekend share -- and how long the longest focused sessions last."
	// rhythmHowToRead is Result.HowToRead for this validator -- see its doc comment.
	rhythmHowToRead = "When the work happened, described and not graded. On a local store every session is one person's, so a verdict here would be a workload judgement about an individual -- which assaio does not make. The bands are printed so a reader who knows the schedule can draw their own conclusion."
	// rhythmDayStart and rhythmDayEnd bound the band this metric calls ordinary hours; a
	// session starting outside them, or on a weekend, is counted off-hours. They are assaio's
	// own boundary and nobody's contract, which is why every surface prints them (B178).
	rhythmDayStart = 8
	rhythmDayEnd   = 18
	// rhythmMarathonMinutes is the focused-work length at which a session counts as a
	// marathon -- long enough that both context and attention degrade.
	rhythmMarathonMinutes = 90
	// rhythmMinSessions is the floor below which timing shares are too thin to describe: two
	// evening sessions are a coincidence, not a pattern.
	rhythmMinSessions = 5
)

func init() { Register(rhythmValidator{}) }

// rhythmValidator reads the window's working pattern: when sessions start and how long
// the longest focused stretches run.
type rhythmValidator struct{}

func (rhythmValidator) Name() string       { return rhythmName }
func (rhythmValidator) Title() string      { return rhythmTitle }
func (rhythmValidator) Describe() string   { return rhythmDescribe }
func (rhythmValidator) Layer() layer.Layer { return layer.Activity } // when sessions run

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

	r.Read = rhythmRead(sufficient)
	r.Purity = neutralPurity
	r.Figures = []Figure{
		{Label: "sessions timed", Value: humanize.Int(int64(len(timed)))},
		{Label: "off-hours", Value: humanize.PercentAt(p.OffHoursShare, 0), Note: humanize.PercentAt(p.WeekendShare, 0) + " on weekends, outside " + rhythmDayBand()},
		basisFigure("longest sessions", minutesLabel(p.P95ActiveMinutes)+" p95", "focused work", len(paced)),
		basisFigure("marathons", humanize.PercentAt(p.MarathonShare, 0), "over "+strconv.Itoa(rhythmMarathonMinutes)+" min focused, assaio's own line", len(paced)),
	}
	r.Bars = rhythmBands(p.Bands)
	r.Takeaway = rhythmTakeaway(sufficient, len(paced), p)
	r.Caveats = append(
		r.Caveats,
		"Prov.: hours are read in this machine's local timezone; sessions recorded in another zone land in the wrong band. \"Ordinary hours\" here means "+rhythmDayBand()+" on a weekday, which is assaio's own band and not a claim about anyone's contract.",
		"No verdict on purpose: on a local store every session is one person's, so a good/bad call about when they ran would be a workload judgement about an individual -- the thing this project refuses to make (B178).",
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

// rhythmDayBand renders the hours this metric treats as ordinary, so the boundary a reader is
// being measured against is on the surface rather than in the source.
func rhythmDayBand() string {
	return strconv.Itoa(rhythmDayStart) + ":00-" + strconv.Itoa(rhythmDayEnd) + ":00"
}

// rhythmRead never judges. Off-hours work is a scheduling fact whose meaning belongs to whoever
// knows the schedule -- a timezone assaio read wrong, an on-call week, a preference -- and on a
// local store the subject of any verdict here is one named person.
func rhythmRead(sufficient bool) Read {
	if !sufficient {
		return noDataRead
	}
	return reportedRead
}

// rhythmTakeaway describes and stops. Below the session floor it says the shares are too thin
// to describe at all, which is a different statement from declining to grade them.
func rhythmTakeaway(sufficient bool, paced int, p rhythmProfile) string {
	switch {
	case paced == 0:
		return "No source in this window records focused minutes, so how long sessions run cannot be read; the timing shares above stand on their own."
	case !sufficient:
		return "Too few sessions this window to read a timing pattern -- the shares above are shown as they are."
	default:
		return humanize.PercentAt(p.OffHoursShare, 0) + " of sessions started outside " + rhythmDayBand() +
			" or on a weekend, and " + humanize.PercentAt(p.MarathonShare, 0) + " ran over " +
			strconv.Itoa(rhythmMarathonMinutes) + " focused minutes. What that means about how the work is scheduled is not something a session log knows."
	}
}
