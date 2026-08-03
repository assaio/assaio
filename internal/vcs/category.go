package vcs

import (
	"path"
	"strings"

	"github.com/assaio/assaio/internal/event"
)

// The categories a changed file counts toward. "other" is a real answer, not a fallback for
// source: an image or a binary is neither, and folding it into source would inflate the one
// number a reader is most likely to act on.
const (
	catTest      = "test"
	catSource    = "source"
	catDocs      = "docs"
	catConfig    = "config"
	catGenerated = "generated"
	catOther     = "other"
)

// The rules, in the order they are asked. Order is the whole design: a lock file under
// testdata/ is generated before it is test, and a .yml under docs/ is documentation before
// it is configuration.
var (
	generatedDirs   = []string{"vendor", "node_modules", "dist", "build", "target"}
	generatedNames  = []string{"go.sum", "package-lock.json", "yarn.lock", "Cargo.lock", "poetry.lock"}
	generatedSuffix = []string{".pb.go", "_gen.go", "_generated.go", ".min.js", ".min.css", ".lock"}
	testDirs        = []string{"test", "tests", "testdata", "__tests__", "spec", "fixtures"}
	docDirs         = []string{"doc", "docs"}
	docExts         = []string{".md", ".rst", ".adoc", ".txt"}
	docNames        = []string{"LICENSE", "NOTICE", "AUTHORS", "COPYING"}
	configExts      = []string{".yml", ".yaml", ".json", ".toml", ".ini", ".cfg", ".conf", ".properties", ".env", ".mod"}
	configNames     = []string{"Makefile", "Dockerfile", "Procfile"}
	sourceExts      = []string{
		".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rb", ".java", ".kt", ".rs", ".c", ".h",
		".cc", ".cpp", ".cs", ".php", ".swift", ".scala", ".sh", ".sql", ".css", ".scss",
		".html", ".vue", ".ex", ".exs", ".hs", ".lua", ".pl", ".r", ".proto",
	}
)

// classify names the category a changed path counts toward. p is a git path -- always
// "/"-separated, on every platform -- read by name only, never opened. The answer is
// deliberately a heuristic: a category, not a fact about what the change did. The path is
// used here and nowhere else; no caller stores it.
func classify(p string) string {
	p = path.Clean(p)
	dirs := strings.Split(path.Dir(p), "/")
	name := path.Base(p)
	ext := strings.ToLower(path.Ext(name))

	switch {
	case anyDir(dirs, generatedDirs) || has(generatedNames, name) || hasSuffix(name, generatedSuffix):
		return catGenerated
	case anyDir(dirs, testDirs) || testName(name):
		return catTest
	case anyDir(dirs, docDirs) || has(docExts, ext) || has(docNames, name):
		return catDocs
	case has(configExts, ext) || has(configNames, name) || strings.HasPrefix(name, "."):
		return catConfig
	case has(sourceExts, ext):
		return catSource
	default:
		return catOther
	}
}

// tally counts p into f, so the split can never disagree with the file count beside it.
func tally(f *event.FileCategories, p string) {
	switch classify(p) {
	case catTest:
		f.Test++
	case catSource:
		f.Source++
	case catDocs:
		f.Docs++
	case catConfig:
		f.Config++
	case catGenerated:
		f.Generated++
	default:
		f.Other++
	}
}

// testName covers the three shapes test files take across languages: a suffix (_test.go), an
// infix (app.spec.ts, app.test.ts), and a prefix (test_thing.py).
func testName(name string) bool {
	stem := strings.TrimSuffix(name, path.Ext(name))
	return strings.HasSuffix(stem, "_test") || strings.HasSuffix(stem, ".test") ||
		strings.HasSuffix(stem, ".spec") || strings.HasPrefix(stem, "test_")
}

func anyDir(dirs, want []string) bool {
	for _, d := range dirs {
		if has(want, d) {
			return true
		}
	}
	return false
}

func has(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

func hasSuffix(name string, suffixes []string) bool {
	for _, s := range suffixes {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}
