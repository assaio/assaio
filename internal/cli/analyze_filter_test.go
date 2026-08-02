package cli

import (
	"strings"
	"testing"
)

func TestAnalyzeFilterStatesWhatItNarrowedTo(t *testing.T) {
	seedMarkStore(t)
	if _, err := runCLI(t, "mark", "aaaa1111", "--task", "refactor"); err != nil {
		t.Fatal(err)
	}

	out, err := runCLI(t, "analyze", "--since", "30d", "--task", "refactor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "filtered to task=refactor · 1 of 2 sessions") {
		t.Fatalf("analyze output = %q, want the narrowing stated", out)
	}
	if !strings.Contains(out, "excluded from this view, not from your data") {
		t.Fatalf("analyze output = %q, want unlabeled usage explicitly not written off", out)
	}
	// The verdicts that describe the whole window cannot be restated over a slice of it.
	if !strings.Contains(out, "subscription-fit") {
		t.Fatalf("analyze output = %q, want the window-scoped validators named as skipped", out)
	}
	if strings.Contains(out, "SUBSCRIPTION-FIT · ") {
		t.Fatalf("analyze output = %q, want no window-scoped verdict rendered under a filter", out)
	}
}

func TestAnalyzeFilterMatchingNothingIsNotAnError(t *testing.T) {
	seedMarkStore(t)

	out, err := runCLI(t, "analyze", "--since", "30d", "--task", "docs")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "0 of 2 sessions") || !strings.Contains(out, "nothing is labeled that way yet") {
		t.Fatalf("analyze output = %q, want an honest empty result", out)
	}
}

func TestAnalyzeRejectsUnknownLabelValue(t *testing.T) {
	seedMarkStore(t)

	_, err := runCLI(t, "analyze", "--since", "30d", "--task", "chore")
	if err == nil || !strings.Contains(err.Error(), "unknown --task") {
		t.Fatalf("error = %v, want a vocabulary error rather than an empty result", err)
	}
}

func TestAnalyzeRefusesToFilterAWindowScopedMetric(t *testing.T) {
	seedMarkStore(t)

	_, err := runCLI(t, "analyze", "subscription-fit", "--since", "30d", "--task", "refactor")
	if err == nil || !strings.Contains(err.Error(), "cannot be read per label") {
		t.Fatalf("error = %v, want a refusal to narrow a window-scoped metric", err)
	}
}

func TestReportByTaskAlwaysShowsUnlabeled(t *testing.T) {
	seedMarkStore(t)
	if _, err := runCLI(t, "mark", "aaaa1111", "--task", "refactor"); err != nil {
		t.Fatal(err)
	}

	out, err := runCLI(t, "report", "--since", "30d", "--by", "task")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "refactor") || !strings.Contains(out, "unlabeled") {
		t.Fatalf("report --by task = %q, want both the labeled and the unlabeled group", out)
	}
}

// TestLabelsDoNotMoveUnfilteredFigures is the guarantee the whole feature rests on:
// annotating a session must not change a single number anyone was already reading.
func TestLabelsDoNotMoveUnfilteredFigures(t *testing.T) {
	seedMarkStore(t)

	for _, args := range [][]string{
		{"report", "--since", "30d", "--by", "project"},
		{"effectiveness", "--since", "30d", "--by", "project"},
		{"analyze", "--since", "30d"},
	} {
		t.Run(args[0], func(t *testing.T) {
			before, err := runCLI(t, args...)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := runCLI(t, "mark", "aaaa1111", "--task", "refactor", "--outcome", "done"); err != nil {
				t.Fatal(err)
			}
			after, err := runCLI(t, args...)
			if err != nil {
				t.Fatal(err)
			}
			if withoutIntent(before) != withoutIntent(after) {
				t.Fatalf("%v changed after labeling a session:\n--- before ---\n%s\n--- after ---\n%s", args, before, after)
			}
			if _, err := runCLI(t, "mark", "aaaa1111", "--unmark"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// withoutIntent drops the intent verdict, the one metric whose subject is the labels
// themselves and which is therefore meant to change when a session is labeled. Every other
// block must be byte-identical.
func withoutIntent(out string) string {
	blocks := strings.Split(out, "\n\n")
	kept := blocks[:0]
	for _, b := range blocks {
		if !strings.HasPrefix(strings.TrimSpace(b), "INTENT · ") {
			kept = append(kept, b)
		}
	}
	return strings.Join(kept, "\n\n")
}
