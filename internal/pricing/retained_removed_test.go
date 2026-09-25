package pricing

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// A removal claims a key is gone from retained.json on purpose. The claim must stay true: an id
// a refresh re-added (LiteLLM listed it again) is priced again, and the line would lie.
func TestRetainedRemovalsAreStillRemoved(t *testing.T) {
	text, err := os.ReadFile("retained_removed.txt")
	if err != nil {
		t.Fatal(err)
	}
	removed, err := parseRemovals(string(text))
	if err != nil {
		t.Fatalf("retained_removed.txt: %v", err)
	}
	raw, err := os.ReadFile("retained.json")
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("retained.json: %v", err)
	}
	for id := range removed {
		if _, ok := keys[id]; ok {
			t.Errorf("retained_removed.txt lists %q, but retained.json still holds it; "+
				"delete its line from retained_removed.txt", id)
		}
	}
}

func TestParseRemovals(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    map[string]string
		wantErr string
	}{
		{"header only", "# ids\n# more\n", map[string]string{}, ""},
		{"an entry keeps its whole reason", "# h\n\ngpt-x  dropped by LiteLLM\n", map[string]string{"gpt-x": "dropped by LiteLLM"}, ""},
		{"an indented comment is a comment", "  # gpt-x\n", map[string]string{}, ""},
		{"an id without a reason", "gpt-x\n", nil, "line 1: gpt-x has no reason"},
		{"an id listed twice", "gpt-x a\ngpt-x b\n", nil, "line 2: gpt-x is already listed on line 1"},
		{"a tab separates and a CRLF ending is dropped", "gpt-x\tdropped\r\n", map[string]string{"gpt-x": "dropped"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRemovals(tt.text)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for id, reason := range tt.want {
				if got[id].reason != reason {
					t.Errorf("%s reason = %q, want %q", id, got[id].reason, reason)
				}
			}
		})
	}
}

type removal struct {
	reason string
	line   int
}

// parseRemovals reads retained_removed.txt with the rule the consistency workflow's awk applies:
// fields split on spaces and tabs, and a line whose first field starts with # is a comment.
func parseRemovals(text string) (map[string]removal, error) {
	out := map[string]removal{}
	for i, line := range strings.Split(text, "\n") {
		fields := strings.FieldsFunc(strings.TrimSuffix(line, "\r"), func(r rune) bool { return r == ' ' || r == '\t' })
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		id, n := fields[0], i+1
		if len(fields) < 2 {
			return nil, fmt.Errorf("line %d: %s has no reason; write `%s <why it was removed>`", n, id, id)
		}
		if prev, ok := out[id]; ok {
			return nil, fmt.Errorf("line %d: %s is already listed on line %d; keep one line", n, id, prev.line)
		}
		out[id] = removal{reason: strings.Join(fields[1:], " "), line: n}
	}
	return out, nil
}
