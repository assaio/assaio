package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The guides tell an author which helper to reach for, and a rename leaves that advice behind
// silently: four pages recommended `shareOrDash` for a divide-by-zero and no package had such a
// function. This is the narrow check that catches it -- narrow on purpose. Inferring which
// symbol a sentence means from the nearest file path was tried and produced mostly noise, and a
// guard that gets switched off is worse than an admitted gap.
//
// Each entry is one helper the documentation actively recommends, and the file that must
// declare it. Both directions are checked: an entry naming a symbol that no longer exists
// fails, and so does an entry no document mentions any more.
var recommendedHelpers = map[string]string{
	"PercentOrDash":  "internal/humanize/percent.go",
	"perActiveDay":   "internal/analyze/format.go",
	"readFor":        "internal/analyze/format.go",
	"fracOf":         "internal/analyze/format.go",
	"UsageAnswering": "internal/report/answering.go",
	"TokensIn":       "internal/report/answering.go",
	"Answers":        "internal/parser/depth_query.go",
	"Register":       "internal/analyze/analyze.go",
	"Stamp":          "internal/analyze/confidence.go",
}

// CheckRecommendedHelpers reports a helper the documentation recommends that its file no longer
// declares, and a listed helper the documentation has stopped recommending. corpus is every
// published document, joined.
func CheckRecommendedHelpers(repoRoot, corpus string) []Problem {
	var problems []Problem
	for name, file := range recommendedHelpers {
		body, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(file))) //nolint:gosec // a path this table names, under the repository root
		if err != nil {
			problems = append(problems, Problem{File: file, Text: fmt.Sprintf(
				"is named as the home of `%s`, and the repository has no such file", name,
			)})
			continue
		}
		if !declares(string(body), name) {
			problems = append(problems, Problem{File: file, Text: fmt.Sprintf(
				"no longer declares `%s`, which the documentation still recommends -- update the "+
					"guides and internal/docs/symbols.go together", name,
			)})
		}
		// A bare occurrence counts: inside a fenced example a helper is written as code,
		// not in backticks, and an example is a recommendation like any other.
		if !strings.Contains(corpus, name) {
			problems = append(problems, Problem{File: "docs/", Text: fmt.Sprintf(
				"no longer mentions `%s`, so listing it in internal/docs/symbols.go claims a "+
					"recommendation nothing makes", name,
			)})
		}
	}
	return problems
}

// declares reports whether a Go source declares name at the top level, in any of the forms a
// document refers to one by: a function, a method, or a type.
func declares(body, name string) bool {
	for _, form := range []string{"func " + name + "(", ") " + name + "(", "type " + name + " "} {
		if strings.Contains(body, form) {
			return true
		}
	}
	return false
}
