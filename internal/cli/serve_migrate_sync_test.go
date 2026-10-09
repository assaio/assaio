package cli

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func TestReadSyncMigrationManifest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"one mapping", `{"members":[{"from":"alice","to":"member-v2-11111111111111111111111111111111"}]}`, false},
		{"duplicate old label", `{"members":[{"from":"alice","to":"member-v2-11111111111111111111111111111111"},{"from":"alice","to":"member-v2-22222222222222222222222222222222"}]}`, true},
		{"unknown field", `{"members":[],"secret":"never"}`, true},
		{"two documents", `{"members":[]} {"members":[]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "mapping.json")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			mapping, err := readSyncMigrationManifest(path)
			if (err != nil) != tc.wantErr {
				t.Fatalf("mapping=%+v err=%v, wantErr=%v", mapping, err, tc.wantErr)
			}
			if !tc.wantErr && mapping["alice"] != "member-v2-11111111111111111111111111111111" {
				t.Fatalf("mapping=%+v", mapping)
			}
		})
	}
}

func TestServeMigrateSyncRekeysExistingStore(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dbPath := filepath.Join(t.TempDir(), "server.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	record := usage.Record{
		Tool: "claude-code", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m",
		DedupeKey: "alice:k", Member: "alice", GitBranch: "private/branch",
		Granularity: "turn", InputTokens: 7,
	}
	if _, err := st.InsertSynced(context.Background(), []usage.Record{record}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `UPDATE sync_protocol SET version = 1 WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	mapPath := filepath.Join(t.TempDir(), "mapping.json")
	const digest = "member-v2-11111111111111111111111111111111"
	if err := os.WriteFile(mapPath, []byte(`{"members":[{"from":"alice","to":"`+digest+`"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"serve", "migrate-sync", "--db", dbPath, "--map", mapPath})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "1 usage rows") || strings.Contains(out.String(), mapPath) {
		t.Fatalf("migration output = %q", out.String())
	}
	st, err = store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	version, err := st.SyncProtocolVersion(context.Background())
	if err != nil || version != 2 {
		t.Fatalf("protocol version=%d err=%v", version, err)
	}
	records, err := st.Export(context.Background(), time.Time{})
	if err != nil || len(records) != 1 || records[0].Member != digest || records[0].DedupeKey != digest+":k" || records[0].GitBranch != "" {
		t.Fatalf("migrated records=%+v err=%v", records, err)
	}
}
