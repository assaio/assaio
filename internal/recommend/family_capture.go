package recommend

import (
	"strings"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/humanize"
)

// captureRepairable are the verdicts that answer "could a backfill fix this". They are the
// only ones that call missingCaptureWhen, which is what makes them the ones to ask: their
// `Recorded` is true exactly when the subject's denominator exists among rows from a source
// that records the field, i.e. when the gap is this build's and not the source's. `coverage`
// looks like the natural place to read and is not -- it never sets the field, so asking it
// returns nil on every window.
var captureRepairable = []string{"friction", "explore-produce"}

func init() { Register(captureGap{}) }

// captureGap proposes the cheapest correction assaio can ever recommend: re-read the logs it
// already has with the build it is running now. It fires only on the gap a re-read can close --
// history parsed by an older build -- never on a source that structurally records nothing,
// where a backfill changes nothing and the suggestion would waste the reader's afternoon.
type captureGap struct{}

func (captureGap) Name() string { return "capture-gap" }

func (captureGap) Propose(e *Evidence) []Record {
	var stale []string
	for _, name := range captureRepairable {
		r := e.Result(name)
		if r == nil || r.Confidence.Recorded == nil || !*r.Confidence.Recorded {
			continue
		}
		if r.Confidence.Signal != nil && *r.Confidence.Signal < 1 {
			stale = append(stale, name+" covers "+humanize.Percent(*r.Confidence.Signal)+" of the window it could")
		}
	}
	if len(stale) == 0 {
		return nil
	}
	return []Record{{
		ID:    "capture-gap/window",
		Title: "Re-read the logs you already have: part of this window was parsed by a build that captured less",
		Evidence: append([]string{
			"the runner attributes the gap to the build that read these rows, not to the source: a tool that does record the field ran here",
		}, stale...),
		// The condition is the runner's own, computed over the whole window rather than
		// sampled, so it is as confident as the store is complete.
		Confidence:    analyze.ConfidenceHigh,
		Scope:         "this store's already-ingested history",
		Action:        "Run `assaio-agent backfill --full`, which re-parses every input and restates the rows an older build left incomplete.",
		Prerequisites: []string{"The source's own logs still on disk -- a backfill cannot recover a transcript the tool has deleted"},
		Effect: "The metrics above reach the evidence they were missing. Nothing is invented: a re-read can only " +
			"find what the log already said, and it may move a figure down as well as up.",
		Risks: []string{
			"A full re-read takes minutes on a large store and rewrites stored rows -- including downward, where a corrected rule now reads less than the old one did. `backfill` reports how many.",
		},
		Rollback:     "None needed: it re-derives from the same inputs. A store restored from a copy returns to the previous reading.",
		ReviewWindow: "immediately -- the same window, re-analyzed",
		FollowUp:     "the signal coverage on " + strings.Join(captureRepairable, " and ") + ", which should reach 100%",
		Status:       StatusProposed,
	}}
}
