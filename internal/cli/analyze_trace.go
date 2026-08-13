package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/trace"
)

// readSequenceFacts fills the two Input fields no windowed aggregate can answer: the window's step
// sequences, and how far back the store's own history goes.
//
// The sequences are narrowed to the sessions the rest of the Input describes, so a filtered run's
// detectors answer for the same population its other figures do. Every surface that runs validators
// goes through the one build path this is called from rather than each deciding for itself: a
// `digest` comparing a run that read the sequences against one that did not would report a detector
// appearing and disappearing as movement in the data.
//
// The sequence read costs 2.5s on a 339,000-step store and is tied to the registry rather than to a
// flag: with no validator reading sequences it is skipped entirely. Every shipped build registers
// two, so the guard is what keeps the cost attached to the reason for it, not a saving anyone sees.
func readSequenceFacts(cmd *cobra.Command, st *store.Store, start time.Time, in *analyze.Input,
	sessions []store.SessionRow,
) error {
	// Unwindowed on purpose: a trend has to know whether the span it compares against existed at
	// all, which the window it is read over cannot say (analyze.Trending).
	oldest, err := st.HistoryStart(cmd.Context(), "")
	if err != nil {
		return err
	}
	in.HistoryStart = oldest
	if !analyze.NeedsTrace(analyze.Validators()) {
		return nil
	}
	sequences, err := st.Timelines(cmd.Context(), start)
	if err != nil {
		return err
	}
	set := trace.New(sequences)
	in.Trace = set.ForSessions(sessions)
	return nil
}
