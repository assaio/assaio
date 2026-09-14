package analyze

import "github.com/assaio/assaio/internal/layer"

// Validator is one independently testable, self-describing metric. Name is a stable
// kebab-case slug (e.g. "model-fit") used on the command line and as the JSON key;
// Title is a human label; Describe is a one-line summary for `assaio analyze --list`.
//
// Layer is on this interface rather than left to each Analyze to remember: a field is
// a field one metric will forget, while a method is a compile error. It answers for the
// metric, not for one window, so it takes no Input.
type Validator interface {
	Name() string
	Title() string
	Describe() string
	Layer() layer.Layer
	Analyze(Input) Result
}

// WindowScoped marks a Validator whose answer belongs to the whole queried window and
// cannot be narrowed to one project -- a flat plan price, attribution pooled across every
// project, per-model turn counts. A project-scoped view (the dashboard's drill-down) skips
// these rather than re-running them against a slice of the data: doing so compares a
// window-wide constant against one project's usage and prints a verdict that contradicts
// the window-level one on the same page.
type WindowScoped interface {
	WindowScoped()
}

// ProjectScoped reports whether v can honestly be re-run over a single project's rows.
func ProjectScoped(v Validator) bool {
	_, window := v.(WindowScoped)
	return !window
}
