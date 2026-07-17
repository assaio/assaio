package pricing

import "testing"

func TestNormalizeModel(t *testing.T) {
	cases := map[string]string{
		"claude-opus-4-5-20251101": "claude-opus-4-5",
		"Claude-Opus-4-5":          "claude-opus-4-5",
		"gpt-5.1":                  "gpt-5.1",
		"claude-opus-4-8[1m]":      "claude-opus-4-8",
	}
	for in, want := range cases {
		if got := NormalizeModel(in); got != want {
			t.Errorf("NormalizeModel(%q) = %q want %q", in, got, want)
		}
	}
}
