package usage

import (
	"encoding/json"
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
