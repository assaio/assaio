package vcs

import (
	"testing"

	"github.com/assaio/assaio/internal/event"
)

// The paths below are spelled with forward slashes on every platform because that is what
// classify is given: git normalises numstat paths to "/" regardless of the OS it runs on.
// Building them with filepath.Join would test a shape the function never receives.
func TestClassify(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"internal/store/store.go", catSource},
		{"internal/store/store_test.go", catTest},
		{"internal/parser/testdata/session.jsonl", catTest},
		{"tests/e2e/run.py", catTest},
		{"src/app.spec.ts", catTest},
		{"docs/adr/0007-canonical-event-contract.md", catDocs},
		{"README.md", catDocs},
		{"LICENSE", catDocs},
		{"go.mod", catConfig},
		{".golangci.yml", catConfig},
		{"Makefile", catConfig},
		{".github/workflows/ci.yml", catConfig},
		{"go.sum", catGenerated},
		{"web/package-lock.json", catGenerated},
		{"vendor/acme/lib.go", catGenerated},
		{"api/service.pb.go", catGenerated},
		{"site/index.html", catSource},
		{"assets/logo.png", catOther},
		{"bin/tool", catOther},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := classify(tc.path); got != tc.want {
				t.Errorf("classify(%q) = %s, want %s", tc.path, got, tc.want)
			}
		})
	}
}

// Every changed file lands in exactly one bucket, so the split always adds up to the file
// count it splits -- the invariant the event payload refuses to be built without.
func TestTallyAccountsForEveryFile(t *testing.T) {
	var f event.FileCategories
	paths := []string{
		"internal/vcs/vcs.go",
		"internal/vcs/vcs_test.go",
		"README.md",
		"go.sum",
		"assets/logo.png",
	}
	for _, p := range paths {
		tally(&f, p)
	}
	if got := f.Total(); got != int64(len(paths)) {
		t.Fatalf("categories account for %d of %d files: %+v", got, len(paths), f)
	}
	if f.Source != 1 || f.Test != 1 || f.Docs != 1 || f.Generated != 1 || f.Other != 1 {
		t.Fatalf("split = %+v, want one file in each of five buckets", f)
	}
}
