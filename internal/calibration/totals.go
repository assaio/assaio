package calibration

import "github.com/assaio/assaio/internal/usage"

// Totals is what a trace's records add up to. One shape for every source, so a figure a
// source cannot carry is asserted as absent rather than left unstated.
type Totals struct {
	Records      int   `json:"records"`
	Sessions     int   `json:"sessions"`
	Input        int64 `json:"input_tokens"`
	Output       int64 `json:"output_tokens"`
	CacheRead    int64 `json:"cache_read_tokens"`
	CacheWrite   int64 `json:"cache_write_tokens"`
	CacheWrite1h int64 `json:"cache_write_1h_tokens"`
	Reasoning    int64 `json:"reasoning_tokens"`
	LinesAdded   int64 `json:"lines_added"`
	LinesRemoved int64 `json:"lines_removed"`
	ReworkLines  int64 `json:"rework_lines"`
	Edits        int64 `json:"edits"`
	ToolCalls    int64 `json:"tool_calls"`
	Rejected     int64 `json:"rejected"`
	ToolErrors   int64 `json:"tool_errors"`
	Compactions  int64 `json:"compactions"`
	// The sequence figures (ADR 0012). Steps is how many observations the trace's own order
	// holds; StepOutcomes how many of them state how they ended, which is a different count from
	// how many succeeded; StepTargets how many name a file. A source with no step reading
	// adjudicates all three as absent, which is the claim its depth row makes.
	Steps        int64 `json:"steps"`
	StepOutcomes int64 `json:"step_outcomes"`
	StepTargets  int64 `json:"step_targets"`
	Skipped      int   `json:"skipped"`
}

// figures is the one place a Totals field is named. Diff, FigureNames and the
// derivation requirement all read it, so a field added without a name is a compile-time
// omission rather than a figure that quietly stops being checked.
func figures(t *Totals) []struct {
	Name  string
	Value int64
} {
	return []struct {
		Name  string
		Value int64
	}{
		{"records", int64(t.Records)},
		{"sessions", int64(t.Sessions)},
		{"input_tokens", t.Input},
		{"output_tokens", t.Output},
		{"cache_read_tokens", t.CacheRead},
		{"cache_write_tokens", t.CacheWrite},
		{"cache_write_1h_tokens", t.CacheWrite1h},
		{"reasoning_tokens", t.Reasoning},
		{"lines_added", t.LinesAdded},
		{"lines_removed", t.LinesRemoved},
		{"rework_lines", t.ReworkLines},
		{"edits", t.Edits},
		{"tool_calls", t.ToolCalls},
		{"rejected", t.Rejected},
		{"tool_errors", t.ToolErrors},
		{"compactions", t.Compactions},
		{"steps", t.Steps},
		{"step_outcomes", t.StepOutcomes},
		{"step_targets", t.StepTargets},
		{"skipped", int64(t.Skipped)},
	}
}

// FigureNames lists every figure a trace must adjudicate.
func FigureNames() []string {
	var t Totals
	fs := figures(&t)
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.Name)
	}
	return out
}

// Mismatch is one figure on which the parser and the adjudicated answer disagree.
type Mismatch struct {
	Figure    string
	Got, Want int64
}

// Diff names every figure on which got and want disagree.
func Diff(got, want *Totals) []Mismatch {
	g, w := figures(got), figures(want)
	var out []Mismatch
	for i := range g {
		if g[i].Value != w[i].Value {
			out = append(out, Mismatch{Figure: g[i].Name, Got: g[i].Value, Want: w[i].Value})
		}
	}
	return out
}

// Sum adds records up the way every surface does, so a trace's expected totals are compared
// against the same arithmetic a report performs.
func Sum(recs []usage.Record, steps []usage.Step, skipped int) Totals {
	t := Totals{Records: len(recs), Skipped: skipped}
	for i := range steps {
		t.Steps++
		if steps[i].Outcome != "" {
			t.StepOutcomes++
		}
		if steps[i].TargetRef != 0 {
			t.StepTargets++
		}
	}
	sessions := make(map[string]struct{}, len(recs))
	for i := range recs {
		r := &recs[i]
		sessions[r.SessionID] = struct{}{}
		t.Input += r.InputTokens
		t.Output += r.OutputTokens
		t.CacheRead += r.CacheReadTokens
		t.CacheWrite += r.CacheWriteTokens
		t.CacheWrite1h += r.CacheWrite1hTokens
		t.Reasoning += r.ReasoningTokens
		t.LinesAdded += r.LinesAdded
		t.LinesRemoved += r.LinesRemoved
		t.ReworkLines += r.ReworkLines
		t.Edits += r.Edits
		t.ToolCalls += r.ToolCalls
		t.Rejected += r.Rejected
		t.ToolErrors += r.ToolErrors
		t.Compactions += r.Compactions
	}
	t.Sessions = len(sessions)
	return t
}
