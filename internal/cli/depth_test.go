package cli

import (
	"strings"
	"testing"
)

func TestDoctorReportsSourceDepth(t *testing.T) {
	driftHome(t)
	out, err := runCommand(t, "doctor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "depth:") {
		t.Fatalf("doctor must publish what each source can tell you: %q", out)
	}
	if !strings.Contains(out, "claude-code deep") {
		t.Fatalf("doctor must name the detected source's tier: %q", out)
	}
}

// TestDoctorListsGapsOnlyForDetectedSources keeps the section useful: spelling out what
// Cline cannot tell you on a machine that has no Cline is noise, and noise is what makes a
// reader skip the line that matters.
func TestDoctorListsGapsOnlyForDetectedSources(t *testing.T) {
	driftHome(t)
	out, err := runCommand(t, "doctor")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "recomputed from tokens for cross-tool consistency") {
		t.Fatalf("an undetected source's gaps must stay out of the section: %q", out)
	}
}
