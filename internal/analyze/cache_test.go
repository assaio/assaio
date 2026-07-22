package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestCacheHygieneReportsReuseShare covers the healthy path: when most billed input is
// served from cache, the read share is reported and the takeaway says so.
func TestCacheHygieneReportsReuseShare(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 200, Out: 100, CacheRead: 800, CacheWrite: 100},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, cacheName).Analyze(in)

	// reuse = CacheRead/(CacheRead+Input) = 800/1000 = 80%.
	if !strings.Contains(figureValues(got.Figures), "80%") {
		t.Fatalf("Figures = %q, want an 80%% cache-read share", figureValues(got.Figures))
	}
	if !strings.Contains(got.Takeaway, "served from cache") {
		t.Fatalf("Takeaway = %q, want the healthy-reuse message", got.Takeaway)
	}
}

// TestCacheHygieneFlagsWriteWaste covers the honest waste signal: cache written far more
// than it is read back is called out rather than passed off as efficient.
func TestCacheHygieneFlagsWriteWaste(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 900, Out: 100, CacheRead: 100, CacheWrite: 900},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, cacheName).Analyze(in)

	// reuse = 100/1000 = 10% -> not healthy, and writes far outweigh reads.
	if !strings.Contains(got.Takeaway, "written more than it is reused") {
		t.Fatalf("Takeaway = %q, want the cache-write-waste message", got.Takeaway)
	}
}
