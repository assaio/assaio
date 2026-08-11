package docs

import (
	"fmt"
	"sort"
	"strings"
)

// How a published recipe is held to still working. They are not equally strong, and saying so is
// the point: "every recipe is checked" was true of this cookbook while a third of it was checked
// only by a command-line lookup, which reads as a much bigger promise than it is.
const (
	// Executed means the block is run and its output asserted.
	Executed = "executed"
	// ShapeChecked means it is parsed and its shape held to an interface -- a renamed method
	// fails, a wrong number does not.
	ShapeChecked = "shape-checked"
	// CommandsChecked means every assaio-agent invocation in it names a real command and real
	// flags. Nothing runs it, and nothing checks the surrounding shell.
	CommandsChecked = "commands-checked"
	// Loaded means it is parsed as configuration and rejected if the shape is wrong.
	Loaded = "loaded"
)

// recipeCoverage classifies every marked block in docs/recipes/. A recipe missing from here, or
// listed here and missing from the pages, fails TestEveryRecipeIsClassified -- so the claim each
// page makes about itself cannot drift away from what the suite does.
var recipeCoverage = map[string]string{
	// docs/recipes/label-rules.md -- loaded as config, then run through the rule engine.
	"conventional-branches": Executed,
	"ticket-keys":           Executed,
	"spikes":                Executed,
	"skills-and-agents":     Executed,
	"entrypoints":           Executed,

	// docs/recipes/rule-plugins.md
	"rule-config":   Loaded,
	"withheld":      Executed,
	"named-read":    Executed,
	"cannot-decide": Executed,
	"verify-rule":   CommandsChecked,

	// docs/recipes/extensions.md
	"weekday-split":  ShapeChecked,
	"answers-gated":  ShapeChecked,
	"plugin-weekday": Executed,
	"plugin-answers": Executed,
	"metric-config":  Loaded,
	"verify-metric":  CommandsChecked,

	// docs/recipes/ci-gates.md
	"budget-tokens":      CommandsChecked,
	"budget-cost":        CommandsChecked,
	"pre-push-hook":      CommandsChecked,
	"actions-job":        CommandsChecked,
	"doctor-before-cost": CommandsChecked,

	// docs/recipes/automation.md
	"weekly-loop":       CommandsChecked,
	"digest-to-file":    CommandsChecked,
	"digest-to-webhook": CommandsChecked,
	"freshness":         CommandsChecked,
}

// CheckRecipeCoverage reports a marked recipe with no classification, and a classification with
// no recipe. found is every recipe name the pages carry.
func CheckRecipeCoverage(found []string) []Problem {
	var problems []Problem
	have := map[string]bool{}
	for _, name := range found {
		have[name] = true
		if recipeCoverage[name] == "" {
			problems = append(problems, Problem{File: "docs/recipes/", Text: fmt.Sprintf(
				"marks a recipe %q that internal/docs/coverage.go does not classify -- say how it "+
					"is checked, or the pages promise more than the suite does", name,
			)})
		}
	}
	for name := range recipeCoverage {
		if !have[name] {
			problems = append(problems, Problem{File: "internal/docs/coverage.go", Text: fmt.Sprintf(
				"classifies %q, which no page marks any more", name,
			)})
		}
	}
	return problems
}

// CheckCoverageCounts holds a page's own table to the classification. The counts are the shape
// of claim this repository has been wrong about before -- a figure typed once and left -- so the
// page states them and the build checks them.
func CheckCoverageCounts(file, content string) []Problem {
	var problems []Problem
	for how, n := range counts() {
		if !strings.Contains(content, fmt.Sprintf("| %d |", n)) ||
			!strings.Contains(content, "| "+how+" |") {
			problems = append(problems, Problem{File: file, Text: fmt.Sprintf(
				"does not state that %d recipes are %s, which is what the classification holds", n, how,
			)})
		}
	}
	return problems
}

func counts() map[string]int {
	out := map[string]int{}
	for _, how := range recipeCoverage {
		out[how]++
	}
	return out
}

// CoverageSummary counts the classifications, so a page can state what is true of it rather than
// a rounder claim somebody has to keep honest by hand.
func CoverageSummary() string {
	counted := counts()
	kinds := make([]string, 0, len(counted))
	for how := range counted {
		kinds = append(kinds, how)
	}
	sort.Strings(kinds)
	parts := make([]string, 0, len(kinds))
	for _, how := range kinds {
		parts = append(parts, fmt.Sprintf("%d %s", counted[how], how))
	}
	return strings.Join(parts, ", ")
}
