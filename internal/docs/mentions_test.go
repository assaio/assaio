package docs

import (
	"strings"
	"testing"
)

func TestCheckMentions(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "prose names a source the way prose does",
			content: "It reads Claude Code and Cline.",
		},
		{
			name:    "an identifier spelling counts too",
			content: "sources: claude-code, cline",
		},
		{
			name:    "a source the document has never heard of fails",
			content: "It reads Claude Code.",
			want:    `never mentions "cline"`,
		},
		{
			name:    "a name found inside a longer word is not a mention",
			content: "It reads Claude Code, and declines everything else.",
			want:    `never mentions "cline"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertProblem(t, CheckMentions(fixture(), "llms.txt", tt.content, "sources"), tt.want)
		})
	}
}

func TestCheckIdentifiers(t *testing.T) {
	// usage and windowStart are the fixture's top-level input fields; usage[].day is nested and
	// deliberately outside the set, since requiring "day" would pass on any document at all.
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "backticked fields are documented",
			content: "The envelope carries `usage` rows and a `windowStart` boundary.",
		},
		{
			name:    "a fenced JSON example documents them too",
			content: "```json\n{\"usage\": [], \"windowStart\": \"2026-08-01T00:00:00Z\"}\n```",
		},
		{
			name:    "a field named beside its value in one span still counts",
			content: "Set `usage: []` and `windowStart: 2026-08-01T00:00:00Z`.",
		},
		{
			name:    "the English word is not evidence the field is documented",
			content: "Describes usage over the window, starting at the window start.",
			want:    `never mentions "usage"`,
		},
		{
			name:    "a field named only outside a code span does not count",
			content: "The envelope carries `usage` rows, and a windowStart boundary.",
			want:    `never mentions "windowStart"`,
		},
		{
			name:    "a field the document never learned about fails",
			content: "The envelope carries `usage` rows.",
			want:    `never mentions "windowStart"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertProblem(t, CheckIdentifiers(fixture(), "extending.md", tt.content, "metricInput"), tt.want)
		})
	}
}

func TestAnUnknownSetIsReported(t *testing.T) {
	assertProblem(t, CheckMentions(fixture(), "llms.txt", "", "widgets"), "which is not a set")
}

func assertProblem(t *testing.T, problems []Problem, want string) {
	t.Helper()
	if want == "" {
		if len(problems) > 0 {
			t.Fatalf("expected no problem, got %v", problems)
		}
		return
	}
	for _, p := range problems {
		if strings.Contains(p.Text, want) {
			return
		}
	}
	t.Fatalf("expected a problem containing %q, got %v", want, problems)
}
