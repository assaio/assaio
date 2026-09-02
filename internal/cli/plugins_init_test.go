package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/plugin"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// A printed skeleton nobody runs is a skeleton that has quietly stopped working, and the first
// person to find out is a stranger following the documentation. Every kind in every language is
// generated, built where it needs building, and put through the same boundary the runtime puts
// a real plugin through.
func TestEveryScaffoldConforms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the shell and python skeletons are POSIX programs")
	}
	if testing.Short() {
		t.Skip("builds and spawns nine subprocesses")
	}
	for _, kind := range []string{"parser", "metric", "rule"} {
		for _, lang := range []string{"go", "python", "sh"} {
			t.Run(kind+"/"+lang, func(t *testing.T) {
				command := buildScaffold(t, kind, lang)
				cfg := plugin.Config{Name: "demo", Command: command, Timeout: 20 * time.Second}
				checkScaffold(t, kind, cfg)
			})
		}
	}
}

// buildScaffold writes one skeleton the way the documented command line does -- stdout is the
// program -- and returns an executable path.
func buildScaffold(t *testing.T, kind, lang string) string {
	t.Helper()
	requireInterpreter(t, lang)
	dir := t.TempDir()
	program := scaffoldProgram(t, kind, lang)

	if lang == "go" {
		source := filepath.Join(dir, "main.go")
		writeExecutable(t, source, program, 0o600)
		mod(t, dir, "mod", "init", "demo")
		mod(t, dir, "build", "-o", "demo", ".")
		return filepath.Join(dir, "demo")
	}
	path := filepath.Join(dir, "demo")
	writeExecutable(t, path, program, 0o700)
	return path
}

// scaffoldProgram is stdout of the documented command line, and only stdout: the next steps go
// to stderr precisely so this redirect writes a runnable file.
func scaffoldProgram(t *testing.T, kind, lang string) string {
	t.Helper()
	root := NewRootCmd()
	var program, guidance bytes.Buffer
	root.SetOut(&program)
	root.SetErr(&guidance)
	root.SetArgs([]string{"plugins", "init", "--kind", kind, "--lang", lang, "--name", "demo"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(guidance.String(), "config.yaml") {
		t.Errorf("the next steps do not say where to declare the plugin: %q", guidance.String())
	}
	return program.String()
}

func requireInterpreter(t *testing.T, lang string) {
	t.Helper()
	binary := map[string]string{"go": "go", "python": "python3", "sh": "sh"}[lang]
	if _, err := exec.LookPath(binary); err != nil {
		t.Skipf("%s is not on PATH", binary)
	}
}

func writeExecutable(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func mod(t *testing.T, dir string, args ...string) {
	t.Helper()
	//nolint:gosec // args are this test's own literals
	cmd := exec.CommandContext(t.Context(), "go", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go %v: %v\n%s", args, err, out)
	}
}

// checkScaffold runs the skeleton through the real boundary for its kind. Nothing here inspects
// the output by hand: a scaffold is conforming exactly when the runtime accepts it.
func checkScaffold(t *testing.T, kind string, cfg plugin.Config) {
	t.Helper()
	ctx := context.Background()
	switch kind {
	case "parser":
		_, stats, err := plugin.Run(ctx, cfg)
		if err != nil {
			t.Fatalf("parser skeleton rejected: %v", err)
		}
		// A skeleton that emitted a record would be inventing usage nobody had, so zero is the
		// only honest starting point -- and a skipped line would mean it emitted a bad one.
		if stats.Records != 0 || stats.Skipped != 0 {
			t.Fatalf("parser skeleton emitted %d records and %d skips, want none of either",
				stats.Records, stats.Skipped)
		}
	case "metric":
		in := scaffoldWindow()
		res, err := plugin.RunMetric(ctx, cfg, &in)
		if err != nil {
			t.Fatalf("metric skeleton rejected: %v", err)
		}
		if res.Name != "plugin:demo" || res.Read.Key == "" || res.Layer == "" {
			t.Fatalf("metric skeleton returned %+v, want a stamped, layered verdict", res)
		}
	case "rule":
		alerts, err := plugin.RunRule(ctx, cfg, []analyze.Result{})
		if err != nil {
			t.Fatalf("rule skeleton rejected: %v", err)
		}
		if len(alerts) != 0 {
			t.Fatalf("rule skeleton raised %d alerts over no verdicts", len(alerts))
		}
	}
}

func scaffoldWindow() analyze.Input {
	usage := []store.UsageRow{{Day: "2026-08-12", Tool: "claude-code", Model: "m", In: 10, Out: 20}}
	return analyze.BuildInput(usage, nil, pricing.Table{}, time.Now(), 7*24*time.Hour, analyze.Delegation{})
}
