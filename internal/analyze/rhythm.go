package analyze

import (
	"sort"
	"strconv"

	"github.com/assaio/assaio/internal/store"
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
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}
	p := buildRhythm(timed)
	sufficient := len(timed) >= rhythmMinSessions
	calm := p.OffHoursShare <= rhythmOffHoursCeiling && p.MarathonShare <= rhythmMarathonCeiling
	steady := calm && sufficient

	r.Read = rhythmRead(sufficient, steady)
	r.Purity = rhythmPurity(p, sufficient)
	r.Figures = []Figure{
		{Label: "sessions timed", Value: strconv.Itoa(len(timed))},
		{Label: "off-hours", Value: formatPercent(p.OffHoursShare, 0), Note: formatPercent(p.WeekendShare, 0) + " on weekends"},
		{Label: "longest sessions", Value: minutesLabel(p.P95ActiveMinutes) + " p95", Note: "focused work"},
		{Label: "marathons", Value: formatPercent(p.MarathonShare, 0), Note: "over " + strconv.Itoa(rhythmMarathonMinutes) + " min focused"},
	}
	r.Bars = rhythmBands(p.Bands)
	r.Takeaway = rhythmTakeaway(steady, sufficient)
	r.Caveats = append(
		r.Caveats,
		"Prov.: hours are read in this machine's local timezone; sessions recorded in another zone land in the wrong band.",
		"Aggregate workload signal -- not an input to evaluating an individual.",
	)
	return r
}

// rhythmProfile is the window's timing picture: how much work fell outside ordinary
// hours, and how long the longest focused sessions ran.
type rhythmProfile struct {
	OffHoursShare    float64
	WeekendShare     float64
	MarathonShare    float64
	P95ActiveMinutes float64
	// Bands counts session starts per time-of-day band, indexed by rhythmBandOrder.
	Bands map[string]int
}

// rhythmBandOrder is the chronological order the time-of-day bands render in, so the bars
// read as a day rather than as a ranking.
var rhythmBandOrder = []string{"night", "morning", "afternoon", "evening"}

// timedSessions keeps only sessions carrying a start timestamp; a zero FirstTs cannot be
// placed in a band and would otherwise silently count as midnight.
func timedSessions(sessions []store.SessionRow) []store.SessionRow {
	out := make([]store.SessionRow, 0, len(sessions))
	for i := range sessions {
		if !sessions[i].FirstTs.IsZero() {
			out = append(out, sessions[i])
		}
	}
	return out
}

// buildRhythm computes the window's timing shares. Session starts are read in the
// reporting machine's local timezone -- the developer's own working day -- which the
// Result caveats, since a team spanning zones needs the server stage to do this right.
func buildRhythm(sessions []store.SessionRow) rhythmProfile {
	p := rhythmProfile{Bands: make(map[string]int, len(rhythmBandOrder))}
	actives := make([]float64, 0, len(sessions))
	var offHours, weekend, marathons int
	for i := range sessions {
		start := sessions[i].FirstTs.Local()
		hour := start.Hour()
		isWeekend := start.Weekday() == 0 || start.Weekday() == 6
		if isWeekend {
			weekend++
		}
		if isWeekend || hour < rhythmDayStart || hour >= rhythmDayEnd {
			offHours++
		}
		p.Bands[bandForHour(hour)]++
		actives = append(actives, sessions[i].ActiveMinutes)
		if sessions[i].ActiveMinutes >= rhythmMarathonMinutes {
			marathons++
		}
	}
	n := float64(len(sessions))
	p.OffHoursShare = float64(offHours) / n
	p.WeekendShare = float64(weekend) / n
	p.MarathonShare = float64(marathons) / n
	sort.Float64s(actives)
	p.P95ActiveMinutes = percentileAt(actives, 0.95)
	return p
}

// bandForHour maps a local hour to its time-of-day band.
func bandForHour(hour int) string {
	switch {
	case hour < rhythmDayStart:
		return "night"
	case hour < 12:
		return "morning"
	case hour < rhythmDayEnd:
		return "afternoon"
	default:
		return "evening"
	}
}

// rhythmBands renders session starts per time-of-day band in chronological order, so the
// bars show the shape of a working day rather than a league table.
func rhythmBands(bands map[string]int) []Bar {
	var maxCount int
	for _, n := range bands {
		if n > maxCount {
			maxCount = n
		}
	}
	out := make([]Bar, 0, len(rhythmBandOrder))
	for _, name := range rhythmBandOrder {
		n := bands[name]
		out = append(out, Bar{
			Label: name,
			Value: strconv.Itoa(n) + " sessions",
			Frac:  fracOf(int64(n), int64(maxCount)),
		})
	}
	return out
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
func rhythmTakeaway(steady, sufficient bool) string {
	switch {
	case !sufficient:
		return "Too few sessions this window to call the rhythm -- the timing shares above are shown without a verdict."
	case steady:
		return "Sessions sit inside ordinary hours and stay a reasonable length."
	default:
		return "A large share of sessions runs off-hours or very long -- worth looking at how the work is scheduled."
	}
}
