package docs

import "github.com/assaio/assaio/internal/analyze"

// Validator is one shipped metric read. Scope is published because a window-scoped read cannot
// honestly be re-run over a single project, and a reference that omitted it would invite exactly
// that -- the dashboard's project drill-down skips them for the same reason.
type Validator struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Describe string `json:"describe"`
	Scope    string `json:"scope"`
}

// Scope values. Both are facts about where a read is honest, not preferences.
const (
	ScopeProject = "project"
	ScopeWindow  = "window"
)

func validators() []Validator {
	registered := analyze.Validators()
	out := make([]Validator, 0, len(registered))
	for _, v := range registered {
		scope := ScopeWindow
		if analyze.ProjectScoped(v) {
			scope = ScopeProject
		}
		out = append(out, Validator{
			Name:     v.Name(),
			Title:    v.Title(),
			Describe: v.Describe(),
			Scope:    scope,
		})
	}
	return out
}
