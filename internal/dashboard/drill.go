package dashboard

import (
	"sort"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// ProjectDrill is the bounded "Assay · <project>" section: the top project by AI lines
// in the queried window, re-assayed on its own rows only, broken down by repository
// subpath.
type ProjectDrill struct {
	Name     string
	Lines    int64
	Sessions int
	Verdicts []analyze.Result
	Subpaths []store.SubpathRow
}

// TopProject returns the project with the most AI-added lines in usage, or "" when usage
// carries no named project. Deterministic: ties break on the lexicographically smaller
// name, never on Go's randomized map iteration order. Exported so the CLI can fetch its
// Store.Subpaths before calling Build, which re-derives the same name internally to
// scope Drill's own Verdicts.
func TopProject(usage []store.UsageRow) string {
	totals := make(map[string]int64)
	for i := range usage {
		if usage[i].Project == "" {
			continue
		}
		totals[usage[i].Project] += usage[i].LinesAdded
	}
	names := make([]string, 0, len(totals))
	for name := range totals {
		names = append(names, name)
	}
	sort.Strings(names)

	top := ""
	var max int64 = -1
	for _, name := range names {
		if totals[name] > max {
			max, top = totals[name], name
		}
	}
	return top
}

// buildDrill assembles the top project's drill-down, or nil when the window has no named
// project. subpaths must already be scoped by the caller to TopProject(in.Usage).
//
//nolint:gocritic // Input is a small value bundle threaded through the validator framework; buildDrill receives it the same way Build does.
func buildDrill(in analyze.Input, subpaths []store.SubpathRow, anonymize bool) *ProjectDrill {
	name := TopProject(in.Usage)
	if name == "" {
		return nil
	}

	scoped := analyze.BuildInput(
		filterUsageByProject(in.Usage, name),
		filterSessionsByProject(in.Sessions, name),
		in.Prices, in.Now, in.Recent,
		// The store has no project-scoped delegation query today, so the drill's
		// model-fit verdict reports the same global sub-agent delegation share as the
		// top-level one -- a known, documented approximation, not a per-project figure.
		in.Delegation,
	)
	verdicts := runValidators(&scoped)
	if anonymize {
		anonymizeVerdicts(verdicts)
	}

	if anonymize {
		name = report.Pseudonym("project", name)
	}
	return &ProjectDrill{
		Name: name, Lines: scoped.Totals.Lines, Sessions: len(scoped.Sessions),
		Verdicts: verdicts, Subpaths: subpaths,
	}
}

func filterUsageByProject(rows []store.UsageRow, project string) []store.UsageRow {
	out := make([]store.UsageRow, 0, len(rows))
	for i := range rows {
		if rows[i].Project == project {
			out = append(out, rows[i])
		}
	}
	return out
}

func filterSessionsByProject(rows []store.SessionRow, project string) []store.SessionRow {
	out := make([]store.SessionRow, 0, len(rows))
	for i := range rows {
		if rows[i].Project == project {
			out = append(out, rows[i])
		}
	}
	return out
}
