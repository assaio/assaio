package pricing

import (
	"os"
	"regexp"
	"testing"
)

var snapshotDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func TestInfo(t *testing.T) {
	models, retained, snapshotDate, err := Info()
	if err != nil {
		t.Fatal(err)
	}
	if models <= 1000 {
		t.Fatalf("Info() models = %d, want > 1000", models)
	}
	f, err := os.Open("litellm.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	_, listed, err := parse(f)
	if err != nil {
		t.Fatal(err)
	}
	onlyKept := 0
	for name := range readTable(t, "retained.json") {
		if !listed[name] {
			onlyKept++
		}
	}
	if retained != onlyKept {
		t.Fatalf("Info() retained = %d, want the %d keys only retained.json prices", retained, onlyKept)
	}
	if snapshotDate != SnapshotDate {
		t.Fatalf("Info() snapshotDate = %q, want %q", snapshotDate, SnapshotDate)
	}
	if !snapshotDatePattern.MatchString(snapshotDate) {
		t.Fatalf("SnapshotDate = %q, want format YYYY-MM-DD", snapshotDate)
	}
}
