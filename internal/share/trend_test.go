package share

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
)

// unwrapped undoes the text card's box and word wrap, so a quoted note is found whole however
// the card broke it across lines.
func unwrapped(card string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(card, "│", " ")), " ")
}

func evaluated(in *analyze.Input) []analyze.Result {
	results := make([]analyze.Result, 0)
	for _, v := range analyze.Validators() {
		results = append(results, analyze.Evaluate(v, in))
	}
	return results
}

// The block is the two figures analyze published, quoted whole on every surface that leaves the
// machine: the text card, the post and the payload the canvas draws from.
func TestTrendQuotesBothFiguresVerbatim(t *testing.T) {
	a, in := buildSample(t)
	v := index(evaluated(&in))
	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(a); err != nil {
		t.Fatalf("encode: %v", err)
	}
	for _, want := range []struct{ validator, label string }{
		{"burn-anomaly", "week-over-week tokens"},
		{"throughput", "week-over-week AI lines"},
	} {
		f, ok := v.figure(want.validator, want.label)
		if !ok || f.Value == "—" {
			t.Fatalf("%s/%s = %+v, want a readable figure for the sample to quote", want.validator, want.label, f)
		}
		for name, body := range map[string]string{"text": unwrapped(Text(&a)), "post": a.Post, "payload": payload.String()} {
			if !strings.Contains(body, f.Value) || !strings.Contains(body, f.Note) {
				t.Errorf("%s does not quote %s = %q · %q", name, want.label, f.Value, f.Note)
			}
		}
	}
	if a.Trend.Tokens.Layer != "activity" || a.Trend.Lines.Layer != "output" {
		t.Errorf("layers = %q / %q, want activity / output", a.Trend.Tokens.Layer, a.Trend.Lines.Layer)
	}
}

// Each is a basis the card cannot vouch for in public, so it publishes no movement on it -- even
// where analyze, which states the same fact beside its figure, still does.
func TestTrendOmissions(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(*analyze.Input, *Basis)
		want  string
		wants bool
	}{
		{"a fresh store publishes the movement", func(*analyze.Input, *Basis) {}, "", true},
		{"sample data", func(_ *analyze.Input, b *Basis) { b.Sample = true }, "sample data cannot support a direction", false},
		{"history start unknown", func(in *analyze.Input, _ *Basis) { in.HistoryStart = time.Time{} }, "store history could not be read", false},
		{"build unknown", func(in *analyze.Input, _ *Basis) { in.ParsedBy = "" }, "parser build for this data is not recorded", false},
		{"more than one build", func(in *analyze.Input, _ *Basis) { in.ParsedBy = "mixed (2 builds)" }, "more than one parser build read this data; a change may be a correction", false},
		{"last read unknown", func(_ *analyze.Input, b *Basis) { b.ReadThrough = time.Time{} }, "time of the last store read is unknown", false},
		{"not read since the weeks ended", func(in *analyze.Input, b *Basis) { b.ReadThrough = in.Now.AddDate(0, 0, -2) }, "at least one source was last read Aug 15, before the recent week ended", false},
		{"read at the moment the recent week ended", func(in *analyze.Input, b *Basis) { b.ReadThrough = analyze.TrendCloses(in) }, "", true},
		{"a source stopped being found", func(_ *analyze.Input, b *Basis) { b.StoppedReading = true }, "a source stopped being found; its rows end where the reading ended", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := sampleInput(t)
			basis := Basis{ReadThrough: in.Now}
			c.edit(&in, &basis)
			a := Build(in, evaluated(&in), "last 30 days", basis)
			if a.Trend.Withheld != c.want {
				t.Fatalf("Withheld = %q, want %q", a.Trend.Withheld, c.want)
			}
			published := a.Trend.Tokens.Value != "—" || a.Trend.Lines.Value != "—"
			if published != c.wants {
				t.Fatalf("moves = %+v / %+v, want published=%v", a.Trend.Tokens, a.Trend.Lines, c.wants)
			}
			if !c.wants && (a.Trend.Tokens.Note != "" || a.Trend.Lines.Note != "" || strings.Contains(a.Post, trendClause)) {
				t.Fatalf("a withheld block still carries a figure: %+v, post %q", a.Trend, a.Post)
			}
			if !strings.Contains(Text(&a), strings.ToUpper(trendTitle)) {
				t.Fatalf("the text card lost the block's title")
			}
		})
	}
}

// A direction beside the achievement half reads as a result, and two directions in one clause
// read as a ratio nobody measured, so each move is its own line and none is on FAME.
func TestTrendStaysOffFameAndOutOfOneClause(t *testing.T) {
	a, _ := buildSample(t)
	var fame, tokens, lines string
	for _, line := range strings.Split(a.Post, "\n") {
		switch {
		case strings.HasPrefix(line, "FAME"):
			fame = line
		case strings.HasPrefix(line, a.Trend.Tokens.Label+" "):
			tokens = line
		case strings.HasPrefix(line, a.Trend.Lines.Label+" "):
			lines = line
		}
	}
	if tokens == "" || lines == "" {
		t.Fatalf("post has no separate line per move:\n%s", a.Post)
	}
	if strings.Contains(fame, a.Trend.Tokens.Note) || strings.Contains(fame, a.Trend.Lines.Note) {
		t.Errorf("FAME carries a movement: %q", fame)
	}
	if strings.Contains(tokens, a.Trend.Lines.Label) {
		t.Errorf("the tokens line also states the lines move: %q", tokens)
	}
}

// A validator that published no trend figure leaves a named absence, never an empty block.
func TestTrendMissingFigureIsNamed(t *testing.T) {
	in := sampleInput(t)
	a := Build(in, nil, "last 30 days", Basis{ReadThrough: in.Now})
	for _, m := range []Move{a.Trend.Tokens, a.Trend.Lines} {
		if m.Value != "—" || m.Note != "not measured in this window" {
			t.Errorf("move = %+v, want a dash and its reason", m)
		}
	}
}
