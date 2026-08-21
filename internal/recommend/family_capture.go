package recommend

import (
	"github.com/assaio/assaio/internal/humanize"
)

// captureMinGap is how much of a window has to be missing capture before re-reading it is
// worth suggesting. Below it the gap is disclosed on every verdict already and a re-parse buys
// a rounding error.
const captureMinGap = 0.1

func init() { Register(captureGap{}) }

// captureGap proposes the cheapest correction assaio can ever recommend: re-read the logs it
// already has with the build it is running now. It only fires on the gap a re-read can close --
// history parsed by an older build -- never on a source that structurally records nothing,
// where a backfill changes nothing and the suggestion would waste the reader's afternoon.
type captureGap struct{}

func (captureGap) Name() string { return "capture-gap" }

func (captureGap) Propose(e *Evidence) []Record {
	coverage := e.Result("coverage")
	if coverage == nil {
		return nil
	}
	// Recorded is the runner's own answer to "could a backfill fix this": it is set only when
	// the missing capture is capture this build never wrote, as opposed to a subject the source
	// does not record. Absent means the question was never asked, which is not a yes.
	if coverage.Confidence.Recorded == nil || *coverage.Confidence.Recorded {
		return nil
	}
	gap := 1 - coverage.Confidence.Activity
	if gap < captureMinGap {
		return nil
	}
	return []Record{{
		ID:    "capture-gap/window",
		Title: "Re-read the logs you already have: part of this window was parsed by a build that captured less",
		Evidence: []string{
			humanize.Percent(gap) + " of the window carries no activity capture, and the runner attributes that to the build that read it rather than to the source",
			"read on " + humanize.Int(int64(coverage.Confidence.Samples)) + " " + coverage.Confidence.Unit,
		},
		Confidence:    coverage.Confidence.Label,
		Scope:         "this store's already-ingested history",
		Action:        "Run `assaio-agent backfill --full`, which re-parses every input and restates the rows an older build left incomplete.",
		Prerequisites: []string{"The source's own logs still on disk -- a backfill cannot recover a transcript the tool has deleted"},
		Effect: "Activity coverage rises toward what the sources actually recorded, and every verdict resting on it gains " +
			"the evidence it was missing. Nothing is invented: a re-read can only find what the log already said.",
		Risks: []string{
			"A full re-read takes minutes on a large store and rewrites stored rows -- including downward, where a corrected rule now reads less than the old one did. `backfill` reports how many.",
		},
		Rollback:     "None needed: it re-derives from the same inputs. A store restored from a copy returns to the previous reading.",
		ReviewWindow: "immediately -- the same window, re-analyzed",
		FollowUp:     "activity coverage on the confidence envelope of any verdict on this window",
		Status:       StatusProposed,
	}}
}
