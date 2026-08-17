package share

import (
	"time"

	"github.com/assaio/assaio/internal/analyze"
)

// num is a figure that may not exist in this window. The bool is not a nicety: a missing
// off-hours share and an off-hours share of zero are different facts, and a hook that
// fired on the second when it meant the first would publish a lie.
//
// s is the string the validator itself published ("92.6%", ">99%"). Copy renders s and
// never re-formats v, because re-rounding a published 92.6% into "93%" is exactly how a
// shared card starts disagreeing with the report it came from.
type num struct {
	v  float64
	ok bool
	s  string
}

// text is the published form, or "" when this window has no such figure. Empty rather than a
// placeholder: a card is a public artifact, and "? of sessions without an edit" is worse than
// a clause that simply is not there.
func (n num) text() string {
	if n.ok {
		return n.s
	}
	return ""
}

// or is the numeric value, or def when absent. It is for arithmetic and thresholds only;
// anything a reader sees goes through text so the card quotes what analyze published.
func (n num) or(def float64) float64 {
	if n.ok {
		return n.v
	}
	return def
}

// facts is the typed reading of one window. Counts come from the prepared aggregates the
// validators themselves read; shares come from the validator that publishes them, parsed
// from its own published string, so no share here is a second opinion about a number
// analyze already answered.
type facts struct {
	tokens int64
	// lines is published only when throughput published it. That validator gates the total on
	// whether any source in the window records a line at all, and says why: summing anyway
	// printed "AI lines total: 0" under an output-layer read, which claims the AI produced
	// nothing rather than that nothing here counts what it produced.
	lines   int64
	linesOK bool
	cost    *float64
	// pricedCoverage is coverage's published, token-weighted share of usage the price table
	// could cost. Totals.Priced is the wrong test: it goes false on a single row of a model
	// with no price even when that model carried no tokens -- the ordinary state, because
	// Claude Code's "<synthetic>" marker is exactly that, and report says so in its own
	// footnote ("they carry no tokens, so the total above is complete"). Reading the boolean
	// printed a floor warning over a complete total.
	pricedCoverage string
	planCost       float64
	sessions       int64
	activeDays     int64
	models         int
	tools          int
	toolCalls      int64
	rejections     int64
	historyDay     float64

	sessionsPerDay num
	conversational num
	commands       num
	delegation     num
	offHours       num
	weekend        num
	rework         num
	premium        num
	cacheRead      num
	compaction     num

	multiple string
	// planPaysOff is subscription-fit's own verdict, not a threshold re-derived here: below 1x
	// the multiple has to travel with the direction, and that validator already decided which
	// side of it this window is on.
	planPaysOff bool
	vsAPI       string
	apiEquiv    string
	perHundred  string
	focusedP95  string
}

func gather(in *analyze.Input, v verdicts) facts {
	f := facts{
		tokens:   in.Totals.Tokens,
		cost:     in.Totals.Cost,
		planCost: in.PlanMonthlyCost,
		sessions: int64(len(in.Sessions)),
		models:   countModels(in.ByModel),
	}
	f.tools = distinctTools(in.Usage)
	if n, ok := v.count("adoption", "active days"); ok {
		f.activeDays = n
	}
	if n, ok := v.count("throughput", "AI lines total"); ok {
		f.lines, f.linesOK = n, true
	}
	if n, ok := v.count("explore-produce", "classified calls"); ok {
		f.toolCalls = n
	}
	f.rejections = rejectionCount(in.Usage)
	f.historyDay = historyDays(in)

	f.sessionsPerDay = parseNum(v.value("adoption", "sessions/active-day"))
	f.conversational = shareOf(v, "session-taxonomy", "conversational")
	f.commands = commandShare(in.Usage)
	f.delegation = shareOf(v, "model-fit", "sub-agent delegation")
	f.offHours = shareOf(v, "rhythm", "off-hours")
	f.weekend = weekendShare(v)
	f.rework = shareOf(v, "rework", "rework")
	f.premium = shareOf(v, "model-fit", "premium (>=$20/1M out)")
	f.cacheRead = shareOf(v, "cache-hygiene", "cache-read share")
	f.compaction = shareOf(v, "context", "compaction rate")

	f.pricedCoverage = v.value("coverage", "priced coverage")
	f.multiple = v.value("subscription-fit", "value multiple")
	f.apiEquiv = v.value("subscription-fit", "API-equivalent")
	f.vsAPI = v.value("subscription-fit", "vs API")
	if r, ok := v["subscription-fit"]; ok {
		f.planPaysOff = r.Read.Key == "good"
	}
	f.focusedP95 = v.value("rhythm", "longest sessions")
	return f
}

// countModels counts what the card will draw, which is models that carried tokens; see
// withTokens.
func countModels(byModel []analyze.ModelStat) int { return len(withTokens(byModel)) }

func shareOf(v verdicts, validator, label string) num {
	got, ok := v.share(validator, label)
	f, _ := v.figure(validator, label)
	return num{v: got, ok: ok, s: f.Value}
}

// weekendShare reads the weekend split out of the off-hours note ("24% on weekends"),
// which is where rhythm publishes it rather than as a figure of its own.
func weekendShare(v verdicts) num {
	f, ok := v.figure("rhythm", "off-hours")
	if !ok {
		return num{}
	}
	return parseLeadingPercent(f.Note)
}

// historyDays is how many days of history the store holds behind this window, which is
// what tells a first run from a long-standing one. Zero when the caller could not answer.
func historyDays(in *analyze.Input) float64 {
	if in.HistoryStart.IsZero() || in.Now.IsZero() {
		return 0
	}
	d := in.Now.Sub(in.HistoryStart)
	if d < 0 {
		return 0
	}
	return d.Hours() / 24
}

// costIsComplete is true only when coverage published a full 100%. Anything short of it --
// including ">99%" -- means some usage carried tokens the table could not cost, and the
// figures over it are floors.
func (f *facts) costIsComplete() bool { return f.pricedCoverage == "100%" }

// windowLayer names the layers the card's own figures rest on. Tokens and cost are
// activity; lines are output. Nothing here reaches outcome, and the card says so.
const windowLayer = "activity + output"

// freshWindow is how much history makes a run a first run rather than a habit. Below it
// the universal day-one hook is the honest opening, because a 30-day frame over two days
// of data would name a window the store cannot fill.
const freshWindow = 7 * 24 * time.Hour
