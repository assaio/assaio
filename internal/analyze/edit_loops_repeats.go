package analyze

import (
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// repeatEdits is what one sequence did to the files it touched.
type repeatEdits struct {
	Project string
	// Edits counts edit steps that named a file. Untargeted counts the ones that did not and
	// therefore cannot be paired with anything -- a coverage fact, never a sequence that behaved
	// well.
	Edits, Untargeted int
	// Repeats is how many edits returned to a file this sequence had already edited, with at
	// least one command step in between.
	Repeats int
}

// Rate is the share of this sequence's targeted edits that returned to an earlier file.
func (r repeatEdits) Rate() float64 { return shareOf(int64(r.Repeats), int64(r.Edits)) }

// countRepeatEdits walks one sequence in order, counting the edits that came back to a file after
// something ran.
//
// The rule is CodeBurn's published one -- "a retry is when the same file is re-edited after a
// shell command in between (Edit foo.ts, Bash, Edit foo.ts). Editing different files across shell
// steps is not a retry" (github.com/getagentseal/codeburn) -- adopted rather than invented, so the
// figure means the same thing as a tool that already ships it. assaio applies it per sequence: a
// sub-agent's edits are counted against that sub-agent, never against the session that launched it.
//
// The command counter is what makes this order-sensitive without a nested scan: an edit repeats a
// file only when more commands have run by now than had run at that file's previous edit.
func countRepeatEdits(t *store.Timeline) repeatEdits {
	out := repeatEdits{Project: t.Project}
	var commands int
	at := make(map[int64]int)
	for i := range t.Steps {
		switch t.Steps[i].Kind {
		case usage.StepCommand:
			commands++
		case usage.StepEdit:
			ref := t.Steps[i].TargetRef
			if ref == 0 {
				out.Untargeted++
				continue
			}
			out.Edits++
			if before, seen := at[ref]; seen && commands > before {
				out.Repeats++
			}
			at[ref] = commands
		}
	}
	return out
}
