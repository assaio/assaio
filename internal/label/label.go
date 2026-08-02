// Package label holds the closed vocabularies for session annotations: the only data in
// assaio a person types rather than a parser extracts. One package, so the mark command,
// the store, the report dimensions and the intent validator cannot drift apart on what a
// valid label is. Category values only -- never free text, never a prompt.
package label

import "strings"

// The annotation axes. Each is a separate question about the same session, so a session
// can carry one, two, or all three.
const (
	Task       = "task"
	Outcome    = "outcome"
	Difficulty = "difficulty"
)

// Unlabeled is the group name usage with no annotation carries in a report grouped by an
// axis. It is always rendered: hiding it would present a slice of the data as the whole.
const Unlabeled = "unlabeled"

// Axes maps each axis to its allowed values, in display order. Adding a value here is the
// only change a new category needs -- deliberately not a SQL CHECK constraint, which would
// make the same edit a migration and a second source of truth.
var Axes = map[string][]string{
	Task:       {"bugfix", "feature", "test", "refactor", "docs", "research", "review", "other"},
	Outcome:    {"done", "partial", "abandoned"},
	Difficulty: {"low", "medium", "high"},
}

// Names lists the axes in the order flags and report dimensions present them; ranging over
// Axes would order them randomly.
var Names = []string{Task, Outcome, Difficulty}

// Valid reports whether v is an allowed value on axis. The empty string is always valid: it
// is how "this axis is not set" is stored, which is what lets a session be marked with a
// task today and an outcome once the work ends.
func Valid(axis, v string) bool {
	if v == "" {
		return true
	}
	for _, allowed := range Axes[axis] {
		if allowed == v {
			return true
		}
	}
	return false
}

// Values renders an axis's vocabulary for a flag description or an error message.
func Values(axis string) string { return strings.Join(Axes[axis], "|") }
