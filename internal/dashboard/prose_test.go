package dashboard

import "testing"

func TestProse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"spaced break becomes an em dash", "a spike nobody noticed -- a runaway loop", "a spike nobody noticed — a runaway loop"},
		{"every occurrence, not just the first", "one -- two -- three", "one — two — three"},
		{"a flag keeps its hyphens", "run `assaio-agent report --anonymize` first", "run `assaio-agent report --anonymize` first"},
		{"an unspaced range is untouched", "2024--2025", "2024--2025"},
		{"text without a break is unchanged", "Usage is broad across projects.", "Usage is broad across projects."},
		{"empty stays empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prose(tt.in); got != tt.want {
				t.Fatalf("prose(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
