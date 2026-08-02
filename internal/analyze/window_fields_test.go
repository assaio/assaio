package analyze

import (
	"encoding/json"
	"testing"

	"github.com/assaio/assaio/internal/store"
)

// Skills, Agents and TurnSizing are filled by the CLI and the server but not by the
// dashboard's project drill, which has no project-scoped query for them. A validator that
// reads one of those fields and is *not* WindowScoped would therefore run in the drill
// against an empty slice and report "no attribution" as a fact about the project -- a
// confident answer to a question nobody asked it.
//
// Today that cannot happen because every such validator is WindowScoped. Nothing enforced it
// until this test: a twentieth validator would break it silently, and the failure would be a
// plausible-looking verdict rather than a crash. ADR 0008.
func TestProjectScopedValidatorsIgnoreWindowOnlyFields(t *testing.T) {
	withFields := favorableInput()
	withFields.Skills = []store.AttributionRow{{Name: "code-review", Tokens: 5000, Lines: 40}}
	withFields.Agents = []store.AttributionRow{{Name: "general-purpose", Tokens: 3000, Lines: 20}}
	withFields.TurnSizing = []store.ModelTurns{{Model: "claude-sonnet-4-5", Turns: 10, SmallTurns: 7}}

	without := favorableInput()

	for _, v := range Validators() {
		if !ProjectScoped(v) {
			continue
		}
		t.Run(v.Name(), func(t *testing.T) {
			withRes, withoutRes := Evaluate(v, &withFields), Evaluate(v, &without)
			a, b := mustJSON(t, &withRes), mustJSON(t, &withoutRes)
			if a != b {
				t.Fatalf("%s reads a window-only field but is not WindowScoped, so the "+
					"project drill would render it against data the caller never supplied:\n"+
					"with=%s\nwithout=%s", v.Name(), a, b)
			}
		})
	}
}

func mustJSON(t *testing.T, r *Result) string {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
