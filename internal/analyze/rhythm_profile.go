package analyze

import (
	"sort"
	"strconv"

	"github.com/assaio/assaio/internal/store"
)

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

// buildRhythm computes the window's timing shares from every timed session, and its length
// shares from paced -- the subset whose source records focused minutes. Session starts are
// read in the reporting machine's local timezone -- the developer's own working day -- which
// the Result caveats, since a team spanning zones needs the server stage to do this right.
func buildRhythm(timed, paced []store.SessionRow) rhythmProfile {
	p := rhythmProfile{Bands: make(map[string]int, len(rhythmBandOrder))}
	var offHours, weekend int
	for i := range timed {
		start := timed[i].FirstTs.Local()
		hour := start.Hour()
		isWeekend := start.Weekday() == 0 || start.Weekday() == 6
		if isWeekend {
			weekend++
		}
		if isWeekend || hour < rhythmDayStart || hour >= rhythmDayEnd {
			offHours++
		}
		p.Bands[bandForHour(hour)]++
	}
	n := float64(len(timed))
	p.OffHoursShare = float64(offHours) / n
	p.WeekendShare = float64(weekend) / n
	p.MarathonShare, p.P95ActiveMinutes = sessionLengths(paced)
	return p
}

// sessionLengths reads the marathon share and the p95 focused stretch off the sessions that
// carry one. Both are zero for an empty subset, which the caller renders as "—" rather than
// as sessions that ended instantly.
func sessionLengths(paced []store.SessionRow) (marathonShare, p95 float64) {
	if len(paced) == 0 {
		return 0, 0
	}
	actives := make([]float64, 0, len(paced))
	var marathons int
	for i := range paced {
		actives = append(actives, paced[i].ActiveMinutes)
		if paced[i].ActiveMinutes >= rhythmMarathonMinutes {
			marathons++
		}
	}
	sort.Float64s(actives)
	return float64(marathons) / float64(len(paced)), percentileAt(actives, 0.95)
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
