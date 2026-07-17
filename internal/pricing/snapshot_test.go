package pricing

import (
	"regexp"
	"testing"
)

var snapshotDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func TestInfo(t *testing.T) {
	models, snapshotDate := Info()
	if models <= 1000 {
		t.Fatalf("Info() models = %d, want > 1000", models)
	}
	if snapshotDate != SnapshotDate {
		t.Fatalf("Info() snapshotDate = %q, want %q", snapshotDate, SnapshotDate)
	}
	if !snapshotDatePattern.MatchString(snapshotDate) {
		t.Fatalf("SnapshotDate = %q, want format YYYY-MM-DD", snapshotDate)
	}
}
