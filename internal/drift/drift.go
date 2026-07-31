// Package drift decides whether a source's ingest numbers stopped looking like
// themselves. Every format assaio parses is vendor-internal, so a tool can change its log
// shape at any time; the failure mode that matters is not a crash but plausible-looking
// under-reporting, which no error path ever reports. These canaries are what turn that
// silence into a warning.
package drift

import (
	"fmt"
	"sort"

	"github.com/assaio/assaio/internal/store"
)

// The canaries, named for what stopped looking right.
const (
	Discovery = "discovery"
	Yield     = "yield"
	Skipped   = "skipped"
	ZeroToken = "zero-token"
)

// Each threshold is paired with a sample floor, and the floor is the important half: a
// share computed from a handful of files or records is not evidence, so below it the
// canary says nothing rather than guess. The two absolute shares sit far above what a
// healthy parse produces -- a real 4.5 GB corpus carries 0.165% zero-token records and no
// skipped lines at all -- so drift has to be gross before either fires.
const (
	discoveryDropShare   = 0.5
	discoveryBaselineMin = 20
	yieldDropShare       = 0.25
	yieldParsedMin       = 20
	skippedMaxShare      = 0.1
	skippedLinesMin      = 50
	zeroTokenMaxShare    = 0.25
	zeroTokenRecordsMin  = 50
)

// Warning is one fired canary: which source, which signal, and the numbers behind it.
type Warning struct {
	Tool   string
	Canary string
	Detail string
}

// canaries runs in the order a warning list reads best: what could not be found, then
// what could not be got out of what was found.
var canaries = []struct {
	name  string
	check func(cur *store.SourceRun, prev []store.SourceRun) (string, bool)
}{
	{Discovery, discoveryCanary},
	{Yield, yieldCanary},
	{Skipped, skippedCanary},
	{ZeroToken, zeroTokenCanary},
}

// Evaluate judges one source's most recent run -- the last element of runs -- against the
// runs before it. History is compared by median rather than against the previous run, so
// one odd pass cannot move the baseline, and every comparison is a ratio, so an
// incremental run that read four files stays comparable to a full one that read six
// thousand.
func Evaluate(runs []store.SourceRun) []Warning {
	if len(runs) == 0 {
		return nil
	}
	cur, prev := &runs[len(runs)-1], runs[:len(runs)-1]
	var out []Warning
	for _, c := range canaries {
		if detail, ok := c.check(cur, prev); ok {
			out = append(out, Warning{Tool: cur.Tool, Canary: c.name, Detail: detail})
		}
	}
	return out
}

// discoveryCanary catches a source that moved, renamed or stopped writing its logs: the
// files were there last time and are not now. Losing all of them needs no sample floor --
// "was some, now none" is unambiguous at any size -- while a partial drop does, since a
// small corpus swings by whole percentage points on one file.
func discoveryCanary(cur *store.SourceRun, prev []store.SourceRun) (string, bool) {
	base := median(samples(prev, func(r store.SourceRun) (float64, bool) {
		return float64(r.Discovered), true
	}))
	if base <= 0 {
		return "", false
	}
	if cur.Discovered == 0 {
		return fmt.Sprintf("found no files, was ~%.0f over the last %d run(s)", base, len(prev)), true
	}
	if base >= discoveryBaselineMin && float64(cur.Discovered) < discoveryDropShare*base {
		return fmt.Sprintf("found %d file(s), was ~%.0f over the last %d run(s)",
			cur.Discovered, base, len(prev)), true
	}
	return "", false
}

// yieldCanary catches a change that still parses as valid JSON but no longer matches the
// usage shape: the files are found and read, and far less comes out of them than history
// says should. A run that read nothing has no yield to judge.
func yieldCanary(cur *store.SourceRun, prev []store.SourceRun) (string, bool) {
	if cur.Parsed < yieldParsedMin {
		return "", false
	}
	base := median(samples(prev, func(r store.SourceRun) (float64, bool) {
		if r.Parsed == 0 {
			return 0, false
		}
		return float64(r.Records) / float64(r.Parsed), true
	}))
	if base <= 0 {
		return "", false
	}
	got := float64(cur.Records) / float64(cur.Parsed)
	if got >= yieldDropShare*base {
		return "", false
	}
	return fmt.Sprintf("%.1f record(s)/file across %d file(s), history is ~%.1f",
		got, cur.Parsed, base), true
}

// skippedCanary catches lines that stopped unmarshaling -- the loud half of format drift,
// which every parser already counts but nothing ever judged.
func skippedCanary(cur *store.SourceRun, _ []store.SourceRun) (string, bool) {
	total := cur.Records + cur.Skipped
	if cur.Skipped < skippedLinesMin || total == 0 {
		return "", false
	}
	share := float64(cur.Skipped) / float64(total)
	if share < skippedMaxShare {
		return "", false
	}
	return fmt.Sprintf("%d of %d line(s) skipped (%.1f%%)", cur.Skipped, total, share*100), true
}

// zeroTokenCanary catches the quiet half: a renamed or moved token field still parses, so
// records keep arriving at the usual rate while the numbers they carry go to zero. This
// is the exact shape of silent under-reporting.
func zeroTokenCanary(cur *store.SourceRun, _ []store.SourceRun) (string, bool) {
	if cur.Records < zeroTokenRecordsMin {
		return "", false
	}
	share := float64(cur.ZeroToken) / float64(cur.Records)
	if share < zeroTokenMaxShare {
		return "", false
	}
	return fmt.Sprintf("%d of %d record(s) carry no tokens (%.1f%%)",
		cur.ZeroToken, cur.Records, share*100), true
}

// samples pulls one comparable number per historical run, dropping the runs that cannot
// contribute one.
func samples(runs []store.SourceRun, of func(store.SourceRun) (float64, bool)) []float64 {
	out := make([]float64, 0, len(runs))
	for _, r := range runs {
		if v, ok := of(r); ok {
			out = append(out, v)
		}
	}
	return out
}

// median is the baseline every history comparison uses: unlike a mean or a
// previous-run comparison, one anomalous pass cannot drag it.
func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	mid := len(vals) / 2
	if len(vals)%2 == 1 {
		return vals[mid]
	}
	return (vals[mid-1] + vals[mid]) / 2
}
