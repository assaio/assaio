package analyze

import "strconv"

const (
	cacheName      = "cache-hygiene"
	cacheTitle     = "Cache Hygiene"
	cacheDescribe  = "Prompt-cache reuse: how much billed input was served from cache vs re-sent, and whether cache writes are being reused."
	cacheHowToRead = "High cache reuse means repeated context is served cheaply from cache instead of re-billed as fresh input. It is a cost signal, not a quality one -- a big one-shot task legitimately shows low reuse, and vendor cache lifetimes are invisible here."
	// cacheGoodReuse is the cache-read share above which reuse reads as healthy.
	cacheGoodReuse = 0.5
)

func init() { Register(cacheValidator{}) }

// cacheValidator reads prompt-cache efficiency: the share of input tokens served from
// cache (cheaper) versus re-sent as fresh input, and whether cache writes are paying off.
type cacheValidator struct{}

func (cacheValidator) Name() string     { return cacheName }
func (cacheValidator) Title() string    { return cacheTitle }
func (cacheValidator) Describe() string { return cacheDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (cacheValidator) Analyze(in Input) Result {
	r := Result{Name: cacheName, Title: cacheTitle, Describe: cacheDescribe, HowToRead: cacheHowToRead}
	t := in.Totals
	if t.Input == 0 && t.CacheRead == 0 && t.CacheWrite == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}

	reuse := t.CacheEfficiency // CacheRead / (CacheRead + Input)
	healthy := reuse >= cacheGoodReuse

	r.Read = readFor(healthy, "Efficient")
	r.Purity = clamp01(reuse)
	r.Figures = []Figure{
		{Label: "cache-read share", Value: honestPercent(reuse), Note: "of billed input"},
		{Label: "cache reads", Value: compactCount(t.CacheRead)},
		{Label: "cache writes", Value: compactCount(t.CacheWrite), Note: cacheWriteNote(t.CacheRead, t.CacheWrite)},
	}
	r.Takeaway = cacheTakeaway(healthy, t.CacheRead, t.CacheWrite)
	r.Caveats = []string{
		"High reuse is cheaper, not better work -- a large one-shot task legitimately shows low reuse.",
		"Vendor cache lifetimes (TTLs) are invisible, so this is a day-grain approximation.",
	}
	return r
}

func cacheTakeaway(healthy bool, read, write int64) string {
	switch {
	case healthy:
		return "Most repeated context is served from cache, keeping input cost down."
	case write > 0 && read < write:
		return "Cache is written more than it is reused -- short or churning sessions may be paying to cache context that is never read back."
	default:
		return "Little input is served from cache this window -- expected for one-shot or exploratory work."
	}
}

// cacheWriteNote flags cache writes that outweigh reads: paying to cache context that is
// not (yet) being read back.
func cacheWriteNote(read, write int64) string {
	if write > 0 && read < write {
		return "written more than reused"
	}
	return ""
}

// compactCount renders a token count compactly, e.g. 33_400_000_000 -> "33.4B",
// 1_500_000 -> "1.5M", 2_300 -> "2.3K". Real cache-read totals reach billions.
func compactCount(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return strconv.FormatFloat(float64(n)/1_000_000_000, 'f', 1, 64) + "B"
	case n >= 1_000_000:
		return strconv.FormatFloat(float64(n)/1_000_000, 'f', 1, 64) + "M"
	case n >= 1_000:
		return strconv.FormatFloat(float64(n)/1_000, 'f', 1, 64) + "K"
	default:
		return strconv.FormatInt(n, 10)
	}
}
