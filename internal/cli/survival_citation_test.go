package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The register adjudicated GitClear's churn figure against survival and refused it (internal/
// threshold), and nothing rendered that refusal: survival printed a bare percentage, which is
// the more natural subtraction of the two -- a survival rate and a churn rate are both
// percentages, and a reader told nothing will take one from the other.
func TestSurvivalNamesTheChurnStudyItIsNotBeingGradedAgainst(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := t.TempDir()
	gitIn(t, dir, "init")
	gitIn(t, dir, "config", "user.email", "t@t.test")
	gitIn(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "app.go"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-m", "c1")

	got := runSurvivalCmd(t, dir)
	for _, want := range []string{
		"Not a line here",
		"GitClear",
		"www.gitclear.com/ai_assistant_code_quality_2025_research",
		"reverted or substantially revised within the subsequent two weeks",
		// The window row is the disqualifying one: survival is monotonic in commit age, and a
		// fixed two-week horizon cannot be age-matched against commits of every age.
		"numerator, denominator, window differ",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("survival output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Line: GitClear") {
		t.Fatalf("survival output draws a line from a study that measures something else:\n%s", got)
	}
}
