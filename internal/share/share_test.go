package share

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// secretProject is a repository name planted in every field a card could conceivably read
// from. No assertion below allows it to appear anywhere in any rendered output.
const secretProject = "acme-internal-billing"

func sampleInput(t *testing.T) analyze.Input {
	t.Helper()
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	var usage []store.UsageRow
	var sessions []store.SessionRow
	for i := range 12 {
		day := now.AddDate(0, 0, -i)
		usage = append(usage,
			store.UsageRow{
				Day: day.Format("2006-01-02"), Tool: "claude-code", Model: "claude-opus-4-8",
				Project: secretProject, Granularity: "turn",
				In: 1000, Out: 20000, CacheRead: 400000, CacheWrite: 9000,
				LinesAdded: 900, Edits: 40, ToolCalls: 200, Rejected: 3, ReworkLines: 60,
				ToolReads: 70, ToolSearches: 10, ToolCommands: 90, ToolWrites: 25, ToolOther: 5,
			},
			store.UsageRow{
				Day: day.Format("2006-01-02"), Tool: "codex", Model: "claude-sonnet-5",
				Project: secretProject + "-web", Granularity: "turn",
				In: 500, Out: 6000, CacheRead: 40000, CacheWrite: 2000, LinesAdded: 300,
			})
		for s := range 4 {
			sessions = append(sessions, store.SessionRow{
				SessionID: "s", Project: secretProject, Tool: "claude-code", Model: "claude-opus-4-8",
				FirstTs: day.Add(time.Duration(s*5) * time.Hour), LastTs: day.Add(time.Duration(s*5+1) * time.Hour),
				Turns: 6, OutputTokens: int64(1000 * (s + 1)), PeakContextTokens: 30000,
				Edits: int64(s), ActiveMinutes: 25,
			})
		}
	}
	prices, err := pricing.Load()
	if err != nil {
		t.Fatalf("pricing.Load: %v", err)
	}
	in := analyze.BuildInput(usage, sessions, prices, now, 7*24*time.Hour, analyze.Delegation{Sub: 300, Total: 2000})
	in.WindowStart = now.AddDate(0, 0, -30)
	in.PlanMonthlyCost = 195
	in.HistoryStart = now.AddDate(0, 0, -40)
	return in
}

func buildSample(t *testing.T) (Assay, analyze.Input) {
	t.Helper()
	in := sampleInput(t)
	results := make([]analyze.Result, 0)
	for _, v := range analyze.Validators() {
		results = append(results, analyze.Evaluate(v, &in))
	}
	return Build(in, results, "last 30 days", false), in
}

// TestNoProjectNameSurvivesAnySurface is the redaction rule as a test rather than as a
// promise: it renders every output the command can produce and fails if a repository name
// reaches any of them. Redaction being structural means there is no flag to get wrong,
// and this is what keeps it that way when a field is added later.
func TestNoProjectNameSurvivesAnySurface(t *testing.T) {
	a, _ := buildSample(t)
	var html bytes.Buffer
	if err := RenderHTML(&html, &a); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	surfaces := map[string]string{
		"html": html.String(),
		"text": Text(&a),
		"post": a.Post,
	}
	for name, body := range surfaces {
		if strings.Contains(body, secretProject) {
			t.Errorf("%s surface leaked the project name %q", name, secretProject)
		}
	}
}

// TestCardQuotesWhatAnalyzePublished is the package's whole contract: a figure on a card
// is the same string the report shows, not a re-rounded copy of it.
func TestCardQuotesWhatAnalyzePublished(t *testing.T) {
	a, in := buildSample(t)
	results := make([]analyze.Result, 0)
	for _, v := range analyze.Validators() {
		results = append(results, analyze.Evaluate(v, &in))
	}
	v := index(results)
	rendered := Text(&a)
	for _, want := range []struct{ validator, label string }{
		{"model-fit", "premium (>=$20/1M out)"},
		{"session-taxonomy", "conversational"},
		{"rework", "rework"},
	} {
		published := v.value(want.validator, want.label)
		if published == "" {
			t.Fatalf("%s/%s published nothing to compare against", want.validator, want.label)
		}
		if !strings.Contains(rendered, published) {
			t.Errorf("card does not quote %s/%s = %q", want.validator, want.label, published)
		}
	}
}

// TestEveryWindowGetsAHookAndAnArchetype covers the promise that no card can render a
// blank where its opening line or its identity should be -- including the empty window a
// first run starts from.
func TestEveryWindowGetsAHookAndAnArchetype(t *testing.T) {
	tests := []struct {
		name   string
		facts  facts
		sample bool
	}{
		{name: "empty window"},
		{name: "sample data", sample: true},
		{name: "first day", facts: facts{sessions: 3, tokens: 412_000, historyDay: 2}},
		{name: "heavy", facts: facts{sessions: 2500, tokens: 42e9, lines: 744_000, activeDays: 26}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.facts
			if h := pick(&f, tt.sample); h.Line == "" {
				t.Error("no hook line")
			}
			if a := classify(&f); a.Name == "" || a.Of != archetypeCount {
				t.Errorf("archetype = %+v", a)
			}
		})
	}
}

// TestProfileMarksExactlyOneReserve is the de-shaming rule expressed as an invariant: a
// card with none would read as "nothing to improve", and a card with several would read
// as a list of faults.
func TestProfileMarksExactlyOneReserve(t *testing.T) {
	tests := []struct {
		name  string
		facts facts
	}{
		{"premium heavy", facts{premium: num{v: 92.6, ok: true, s: "92.6%"}, conversational: num{v: 50, ok: true, s: "50%"}}},
		{"balanced", facts{premium: num{v: 40, ok: true, s: "40%"}, delegation: num{v: 30, ok: true, s: "30%"}}},
		{"only one axis known", facts{rework: num{v: 4, ok: true, s: "4%"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.facts
			p := buildProfile(&f)
			marked := 0
			for _, ax := range p.Axes {
				if ax.Reserve {
					marked++
				}
			}
			if len(p.Axes) > 0 && marked != 1 {
				t.Errorf("marked %d reserves across %d axes, want exactly 1", marked, len(p.Axes))
			}
			if len(p.Axes) > 0 && p.Line == "" {
				t.Error("a marked reserve with no line to show for it")
			}
		})
	}
}
