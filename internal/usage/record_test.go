package usage

import (
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"testing"
)

// TestRecordJSONCarriesNoLocalPath: sync pushes Record as JSON, so the working directory and the
// repository key -- a keyed hash of a path -- must neither be written nor be settable by a body.
func TestRecordJSONCarriesNoLocalPath(t *testing.T) {
	data, err := json.Marshal(Record{Cwd: "/home/me/src/api", RepoKey: "v1:0123abcd", Project: "api"})
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"/home/me/src/api", "v1:0123abcd", "RepoKey", "Cwd"} {
		if strings.Contains(string(data), leak) {
			t.Errorf("marshalled record carries %q: %s", leak, data)
		}
	}
	var r Record
	if err := json.Unmarshal([]byte(`{"Cwd":"/x","RepoKey":"v1:forged"}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.Cwd != "" || r.RepoKey != "" {
		t.Errorf("decoded record = (Cwd %q, RepoKey %q), want both empty", r.Cwd, r.RepoKey)
	}
}

// syncedFields is every key a synced record carries: `sync` sends Record as JSON, whole
// (internal/cli/sync_push.go), so a field added to Record leaves the machine unless it is tagged
// `json:"-"`. Adding one here is a decision ADR 0020 asks to be made on purpose, with PRIVACY.md
// and the threat model's sync row changed in the same commit.
var syncedFields = []string{
	"Agent", "CacheMissReason", "CacheReadTokens", "CacheWrite1hTokens", "CacheWriteTokens",
	"Compactions", "DedupeKey", "Edits", "Entrypoint", "GitBranch", "Granularity", "InputTokens",
	"LinesAdded", "LinesRemoved", "Member", "Model", "OutputTokens", "Project", "ReasoningTokens",
	"Rejected", "ReworkLines", "SessionID", "Sidechain", "Skill", "Subpath", "Timestamp", "Tool",
	"ToolCalls", "ToolCommands", "ToolErrors", "ToolOther", "ToolReads", "ToolSearches",
	"ToolWrites",
}

func TestSyncedFieldsArePinned(t *testing.T) {
	data, err := json.Marshal(Record{})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if !slices.Equal(keys, syncedFields) {
		t.Fatalf("a synced record carries %v, the pinned set is %v", keys, syncedFields)
	}
}
