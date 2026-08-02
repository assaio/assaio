package event

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

var update = flag.Bool("update", false, "update golden files")

func record() usage.Record {
	return usage.Record{
		Tool:             "claude-code",
		SessionID:        "s1",
		Timestamp:        occurred,
		Model:            "claude-opus-5",
		InputTokens:      120,
		OutputTokens:     900,
		CacheReadTokens:  40000,
		CacheWriteTokens: 2000,
		ReasoningTokens:  300,
		DedupeKey:        "turn-1",
		Project:          "assaio",
		Granularity:      "turn",
		LinesAdded:       42,
		LinesRemoved:     7,
		Edits:            3,
		ReworkLines:      2,
	}
}

func TestFromRecordEmitsUsageAndEdit(t *testing.T) {
	r := record()
	got, err := FromRecord(&r, "v0.6.0", observed)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want a usage and an edit observation, got %d", len(got))
	}
	if got[0].Type != TypeUsage || got[1].Type != TypeEdit {
		t.Fatalf("want %s then %s, got %s then %s", TypeUsage, TypeEdit, got[0].Type, got[1].Type)
	}
	if got[0].ID != "turn-1" || got[1].ID != "turn-1#edit" {
		t.Fatalf("ids must derive from the dedupe key, got %q and %q", got[0].ID, got[1].ID)
	}
	u, ok := got[0].Payload.(Usage)
	if !ok {
		t.Fatalf("want a Usage payload, got %T", got[0].Payload)
	}
	if u.InputTokens != 120 || u.ReasoningTokens != 300 {
		t.Fatalf("token fields not carried: %+v", u)
	}
	e, ok := got[1].Payload.(Edit)
	if !ok {
		t.Fatalf("want an Edit payload, got %T", got[1].Payload)
	}
	if e.LinesAdded != 42 || e.ReworkLines != 2 {
		t.Fatalf("activity fields not carried: %+v", e)
	}
	if got[0].Subject.Session != "s1" || got[0].Source.Build != "v0.6.0" {
		t.Fatalf("envelope identity not carried: %+v", got[0])
	}
}

// A source with no edit extraction must produce no edit observation at all, so "no activity
// recorded" stays distinguishable from "no activity happened".
func TestFromRecordOmitsEditWhenThereIsNoActivity(t *testing.T) {
	r := record()
	r.LinesAdded, r.LinesRemoved, r.Edits, r.ReworkLines = 0, 0, 0, 0
	got, err := FromRecord(&r, "v0.6.0", observed)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Type != TypeUsage {
		t.Fatalf("want only a usage observation, got %d: %+v", len(got), got)
	}
}

func TestFromRecordCarriesSessionGrain(t *testing.T) {
	r := record()
	r.Tool, r.Granularity = "copilot-cli", "session"
	got, err := FromRecord(&r, "v0.6.0", observed)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Grain != GrainSession {
		t.Fatalf("session-level data must not masquerade as per-turn, got %q", got[0].Grain)
	}
}

func TestFromRecordIsDeterministic(t *testing.T) {
	r := record()
	first, err := FromRecord(&r, "v0.6.0", observed)
	if err != nil {
		t.Fatal(err)
	}
	second, err := FromRecord(&r, "v0.6.0", observed)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if !bytes.Equal(a, b) {
		t.Fatal("re-observing the same record must produce the same events")
	}
}

func TestFromRecordRejects(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*usage.Record)
		wantErr string
	}{
		{"no dedupe key", func(r *usage.Record) { r.DedupeKey = "" }, "no id"},
		{"no tool", func(r *usage.Record) { r.Tool = "" }, "no source name"},
		{"unknown granularity", func(r *usage.Record) { r.Granularity = "daily" }, "unknown grain"},
		{"no granularity", func(r *usage.Record) { r.Granularity = "" }, "unknown grain"},
		{"no timestamp", func(r *usage.Record) { r.Timestamp = time.Time{} }, "no occurrence time"},
		{"negative tokens", func(r *usage.Record) { r.InputTokens = -1 }, "inputTokens is negative"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := record()
			tc.mutate(&r)
			_, err := FromRecord(&r, "v0.6.0", observed)
			assertErr(t, err, tc.wantErr)
		})
	}
}

// PRIVACY.md promises the working directory is transient and the store holds no path; a
// branch name can name a client. Neither may reach an envelope, whatever the record carries.
func TestFromRecordDropsPathsAndBranches(t *testing.T) {
	r := record()
	r.Cwd = "/Users/someone/work/acme-client-portal"
	r.Subpath = "apps/mobile"
	r.GitBranch = "feature/acme-migration"
	got, err := FromRecord(&r, "v0.6.0", observed)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, leaked := range []string{"acme", "apps/mobile", "Users/someone", "feature/"} {
		if strings.Contains(string(encoded), leaked) {
			t.Fatalf("%q reached the envelope: %s", leaked, encoded)
		}
	}
}

// The golden locks the wire shape: a versioned contract with an encoding nobody pinned is
// not a contract. Regenerate with -update, and treat a diff as a contract change.
func TestFromRecordGoldenEncoding(t *testing.T) {
	r := record()
	events, err := FromRecord(&r, "v0.6.0", observed)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	golden := filepath.Join("testdata", "usage_events.golden.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden) //nolint:gosec // golden is a fixed testdata path, not user input
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("contract encoding changed:\n got=%s\nwant=%s", got, want)
	}
}
