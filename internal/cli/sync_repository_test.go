package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// TestSyncWarnsAboutSharedNamesAndSendsNoRepository: the server learns a project's name and
// nothing about the repository behind it, so sync says when one name stands for two local
// repositories -- and the push itself carries no key.
func TestSyncWarnsAboutSharedNamesAndSendsNoRepository(t *testing.T) {
	setSyncIdentityKey(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	at := time.Now().UTC().Add(-time.Hour)
	rec := func(project, key, id string) usage.Record {
		return usage.Record{
			Tool: "claude-code", SessionID: id, DedupeKey: id, Timestamp: at, Model: "m",
			Project: project, RepoKey: key, Granularity: "turn", InputTokens: 1,
		}
	}
	seedStoreAt(t, evidenceDBPath(t), []usage.Record{
		rec("api", "v1:aaaaaaaa", "1"), rec("api", "v1:bbbbbbbb", "2"), rec("web", "v1:cccccccc", "3"),
	})

	var body []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"protocol":2,"inserted":3,"received":3}`))
	}))
	defer ts.Close()

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sync", "--server", ts.URL, "--token", "t"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "is named api;") || strings.Contains(out.String(), "web") {
		t.Fatalf("output = %q, want a warning naming api and only api", out.String())
	}
	for _, leak := range []string{"v1:", "aaaaaaaa", "RepoKey", "repo_id"} {
		if bytes.Contains(body, []byte(leak)) {
			t.Fatalf("push body carries %q: %s", leak, body)
		}
	}
}
