package cline

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func readFixtureMeta(t *testing.T) taskMetadata {
	t.Helper()
	f, err := os.Open("testdata/task_metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var meta taskMetadata
	if err := json.NewDecoder(f).Decode(&meta); err != nil {
		t.Fatal(err)
	}
	return meta
}

func TestParseGolden(t *testing.T) {
	f, err := os.Open("testdata/ui_messages.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, _, err := ParseTask(f, "task1", readFixtureMeta(t))
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	const golden = "testdata/ui_messages.golden"
	if *update {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch:\n got=%s\nwant=%s", got, want)
	}
}

func TestTokenMappingAndPerRequestModel(t *testing.T) {
	f, err := os.Open("testdata/ui_messages.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, _, err := ParseTask(f, "task1", readFixtureMeta(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records want 2 (non-usage entries must be skipped)", len(recs))
	}

	first := recs[0]
	if first.Tool != tool || first.SessionID != "task1" || first.Granularity != "turn" {
		t.Fatalf("dimensions wrong: %+v", first)
	}
	if first.Model != "claude-sonnet-4-5" {
		t.Fatalf("Model = %q, want claude-sonnet-4-5", first.Model)
	}
	if first.InputTokens != 1200 || first.OutputTokens != 340 ||
		first.CacheWriteTokens != 600 || first.CacheReadTokens != 0 {
		t.Fatalf("token mapping wrong: %+v", first)
	}

	second := recs[1]
	if second.Model != "claude-opus-4-5" {
		t.Fatalf("Model = %q, want claude-opus-4-5 (model switched mid-task)", second.Model)
	}
	if second.InputTokens != 900 || second.OutputTokens != 210 ||
		second.CacheWriteTokens != 0 || second.CacheReadTokens != 1200 {
		t.Fatalf("token mapping wrong: %+v", second)
	}
}

func TestDedupeKeyDeterministic(t *testing.T) {
	meta := readFixtureMeta(t)

	f, err := os.Open("testdata/ui_messages.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	first, _, err := ParseTask(f, "task1", meta)
	if err != nil {
		t.Fatal(err)
	}

	f2, err := os.Open("testdata/ui_messages.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f2.Close() }()
	second, _, err := ParseTask(f2, "task1", meta)
	if err != nil {
		t.Fatal(err)
	}

	if len(first) != len(second) {
		t.Fatalf("record count differs across re-parse: %d vs %d", len(first), len(second))
	}
	want := []string{"task1:0", "task1:1"}
	for i, k := range want {
		if first[i].DedupeKey != k {
			t.Fatalf("DedupeKey[%d] = %q, want %q", i, first[i].DedupeKey, k)
		}
		if first[i].DedupeKey != second[i].DedupeKey {
			t.Fatalf("DedupeKey not deterministic at %d: %q vs %q", i, first[i].DedupeKey, second[i].DedupeKey)
		}
	}
}

func TestSkipsNonUsageEntries(t *testing.T) {
	const log = `[
{"type":"say","say":"task","text":"hello","ts":1751360400000},
{"type":"say","say":"text","text":"thinking...","ts":1751360400100},
{"type":"ask","ask":"followup","text":"{\"tokensIn\":5,\"tokensOut\":5}","ts":1751360400200},
{"type":"say","say":"api_req_started","text":"{\"tokensIn\":100,\"tokensOut\":50,\"cacheWrites\":0,\"cacheReads\":0}","ts":1751360400300}
]`
	recs, _, err := ParseTask(strings.NewReader(log), "t2", taskMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records want 1", len(recs))
	}
	if recs[0].DedupeKey != "t2:0" {
		t.Fatalf("DedupeKey = %q, want t2:0 (index counts emitted records only)", recs[0].DedupeKey)
	}
	if recs[0].Model != "" {
		t.Fatalf("Model = %q, want empty when no model_usage available", recs[0].Model)
	}
}

func TestParseTaskInvalidTextIsSkippedNotError(t *testing.T) {
	const log = `[
{"type":"say","say":"api_req_started","text":"not json","ts":1751360400000},
{"type":"say","say":"api_req_started","text":"{\"tokensIn\":10,\"tokensOut\":5}","ts":1751360400100}
]`
	recs, skipped, err := ParseTask(strings.NewReader(log), "t3", taskMetadata{})
	if err != nil {
		t.Fatalf("malformed api_req_started text must not abort the parse: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d want 1", skipped)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records want 1 (record around the malformed line still parsed)", len(recs))
	}
}

func TestParseDirReadsBothFiles(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "task9")
	if err := os.MkdirAll(taskDir, 0o750); err != nil {
		t.Fatal(err)
	}
	ui := `[{"type":"say","say":"api_req_started","text":"{\"tokensIn\":10,\"tokensOut\":5,\"cacheWrites\":0,\"cacheReads\":0}","ts":1751360400000}]`
	if err := os.WriteFile(filepath.Join(taskDir, "ui_messages.json"), []byte(ui), 0o600); err != nil {
		t.Fatal(err)
	}
	meta := `{"model_usage":[{"ts":1751360400000,"model_id":"claude-haiku-4-5"}]}`
	if err := os.WriteFile(filepath.Join(taskDir, "task_metadata.json"), []byte(meta), 0o600); err != nil {
		t.Fatal(err)
	}

	recs, _, err := ParseDir(taskDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records want 1", len(recs))
	}
	if recs[0].SessionID != "task9" || recs[0].Model != "claude-haiku-4-5" {
		t.Fatalf("record wrong: %+v", recs[0])
	}
}

func TestParseDirWithoutMetadataFileSucceeds(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "task10")
	if err := os.MkdirAll(taskDir, 0o750); err != nil {
		t.Fatal(err)
	}
	ui := `[{"type":"say","say":"api_req_started","text":"{\"tokensIn\":10,\"tokensOut\":5}","ts":1751360400000}]`
	if err := os.WriteFile(filepath.Join(taskDir, "ui_messages.json"), []byte(ui), 0o600); err != nil {
		t.Fatal(err)
	}

	recs, _, err := ParseDir(taskDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Model != "" {
		t.Fatalf("recs = %+v, want 1 record with empty model", recs)
	}
}
