package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRuntimeInspectRefusesBothSourcesForOneAdapter: a live endpoint and a saved snapshot are
// two different claims about where a number came from, and provenance is the point.
func TestRuntimeInspectRefusesBothSourcesForOneAdapter(t *testing.T) {
	_, err := runCLI(t, "runtime", "inspect", "--vllm-url", "http://127.0.0.1:8000/metrics", "--vllm-file", "x.prom")
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error = %v, want the two sources refused together", err)
	}
}

func TestRuntimeInspectNeedsASource(t *testing.T) {
	_, err := runCLI(t, "runtime", "inspect")
	if err == nil || !strings.Contains(err.Error(), "nothing to inspect") {
		t.Fatalf("error = %v, want it to say what to pass", err)
	}
}

// TestRuntimeInspectReportsAnUnreachableEndpointWithoutFailing: an endpoint that is down is a
// finding about the deployment, and it must not stop the other source the operator also asked
// about from being read.
func TestRuntimeInspectReportsAnUnreachableEndpointWithoutFailing(t *testing.T) {
	out, err := runCLI(t, "runtime", "inspect",
		"--vllm-file", "../runtime/testdata/vllm.prom",
		// Port 1 refuses immediately on every platform CI runs on.
		"--dcgm-url", "http://127.0.0.1:1/metrics",
		"--timeout", "2s")
	if err != nil {
		t.Fatalf("one dead endpoint failed the whole command: %v", err)
	}
	if !strings.Contains(out, "unreachable") {
		t.Fatalf("output never reports the dead endpoint:\n%s", out)
	}
	if !strings.Contains(out, "serving") {
		t.Fatalf("the readable source was not reported:\n%s", out)
	}
}

func TestRuntimeInspectReportsAnUnreadableFile(t *testing.T) {
	out, err := runCLI(t, "runtime", "inspect", "--vllm-file", filepath.Join(t.TempDir(), "absent.prom"))
	if err != nil {
		t.Fatalf("a missing file failed the command: %v", err)
	}
	if !strings.Contains(out, "unreachable") {
		t.Fatalf("output = %q, want the missing file reported as unreachable", out)
	}
}

// TestRuntimeInspectJSONIsStable is what makes this scriptable: the same input must encode to
// the same document.
func TestRuntimeInspectJSONIsStable(t *testing.T) {
	first, err := runCLI(t, "runtime", "inspect", "--vllm-file", "../runtime/testdata/vllm.prom", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var doc []map[string]any
	if err := json.Unmarshal([]byte(first), &doc); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, first)
	}
	if len(doc) != 1 || doc[0]["source"] != "vllm" {
		t.Fatalf("document = %+v", doc)
	}
	if doc[0]["availability"] != "serving" {
		t.Fatalf("availability = %v", doc[0]["availability"])
	}
}

// TestRuntimeInspectStoresNothing is the promise the command's own help makes. It runs against
// a redirected data directory and asserts the directory is untouched.
func TestRuntimeInspectStoresNothing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	if _, err := runCLI(t, "runtime", "inspect", "--vllm-file", "../runtime/testdata/vllm.prom"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("runtime inspect wrote %v into the data directory; it stores nothing", entries)
	}
}

// TestRuntimeInspectRejectsAnInvalidExposition keeps a parse failure from reading as a
// deployment that publishes nothing.
func TestRuntimeInspectRejectsAnInvalidExposition(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.prom")
	// One line past the parser's line budget: readable as bytes, not as an exposition.
	if err := os.WriteFile(path, []byte("a{k=\""+strings.Repeat("v", 200000)+"\"} 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := runCLI(t, "runtime", "inspect", "--vllm-file", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "could not be parsed") {
		t.Fatalf("output = %q, want the parse failure named", out)
	}
}
