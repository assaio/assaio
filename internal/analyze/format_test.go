package analyze

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderResultTextHeaderFormat(t *testing.T) {
	var buf bytes.Buffer
	r := Result{Name: "adoption", Title: "Adoption & Usage Breadth", Read: Read{Key: "good", Label: "STRONG"}, Takeaway: "ok"}
	if err := RenderResultText(&buf, r); err != nil {
		t.Fatal(err)
	}
	want := "ADOPTION · Adoption & Usage Breadth  [STRONG]\n"
	if got := buf.String(); !strings.HasPrefix(got, want) {
		t.Fatalf("header = %q, want prefix %q", got, want)
	}
}

// TestRenderResultTextHowToReadLine asserts a non-empty HowToRead renders as a "? "
// line, positioned between the header and the figures.
func TestRenderResultTextHowToReadLine(t *testing.T) {
	var buf bytes.Buffer
	r := Result{
		Name: "x", Title: "X", Read: Read{Label: "OK"},
		HowToRead: "Read this as directional, not a verdict.",
		Figures:   []Figure{{Label: "sessions", Value: "12"}},
		Takeaway:  "done",
	}
	if err := RenderResultText(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "  ? Read this as directional, not a verdict.\n") {
		t.Fatalf("HowToRead line missing, got:\n%s", out)
	}
	headerIdx := strings.Index(out, "[OK]")
	howToIdx := strings.Index(out, "? Read this")
	figureIdx := strings.Index(out, "sessions: 12")
	if headerIdx == -1 || howToIdx == -1 || figureIdx == -1 || headerIdx >= howToIdx || howToIdx >= figureIdx {
		t.Fatalf("want header, then \"? \" line, then figures, got:\n%s", out)
	}
}

// TestRenderResultTextHowToReadOmittedWhenEmpty mirrors the Bars nil/empty distinction:
// an unset HowToRead must not print a stray "? " line.
func TestRenderResultTextHowToReadOmittedWhenEmpty(t *testing.T) {
	var buf bytes.Buffer
	r := Result{Name: "x", Title: "X", Takeaway: "done"}
	if err := RenderResultText(&buf, r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "?") {
		t.Fatalf("empty HowToRead must not render a \"? \" line, got:\n%s", buf.String())
	}
}

func TestRenderResultTextFigureWithAndWithoutNote(t *testing.T) {
	var buf bytes.Buffer
	r := Result{
		Name: "x", Title: "X", Read: Read{Label: "OK"},
		Figures: []Figure{
			{Label: "sessions", Value: "12"},
			{Label: "premium", Value: "50%", Note: "3.3 lines/1M tok"},
		},
		Takeaway: "done",
	}
	if err := RenderResultText(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "  sessions: 12\n") {
		t.Fatalf("no-note figure line missing, got:\n%s", out)
	}
	if !strings.Contains(out, "  premium: 50% (3.3 lines/1M tok)\n") {
		t.Fatalf("with-note figure line missing, got:\n%s", out)
	}
}

// TestRenderResultTextBarsNilVsEmpty asserts the nil/non-nil-empty distinction on Bars:
// nil means "this Result has no Bars section" (nothing printed); a non-nil empty slice
// means "this Result has a Bars section with nothing in it right now" (an honest note).
func TestRenderResultTextBarsNilVsEmpty(t *testing.T) {
	var nilBuf bytes.Buffer
	if err := RenderResultText(&nilBuf, Result{Name: "x", Title: "X", Takeaway: "t"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(nilBuf.String(), "none in this window") {
		t.Fatalf("nil Bars must not print an empty-state line, got:\n%s", nilBuf.String())
	}

	var emptyBuf bytes.Buffer
	r := Result{Name: "x", Title: "X", Bars: []Bar{}, Takeaway: "t"}
	if err := RenderResultText(&emptyBuf, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(emptyBuf.String(), "(none in this window)") {
		t.Fatalf("non-nil empty Bars must print the honest empty-state line, got:\n%s", emptyBuf.String())
	}
}

func TestRenderResultTextBarsRenderLabelValueAndFilledWidth(t *testing.T) {
	var buf bytes.Buffer
	r := Result{
		Name: "x", Title: "X",
		Bars:     []Bar{{Label: "webapp-mono", Value: "72879 lines", Frac: 1}, {Label: "api-service", Value: "1916 lines", Frac: 0}},
		Takeaway: "t",
	}
	if err := RenderResultText(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "webapp-mono: 72879 lines  ["+strings.Repeat("#", barWidth)+"]") {
		t.Fatalf("full-frac bar not fully filled, got:\n%s", out)
	}
	if !strings.Contains(out, "api-service: 1916 lines  ["+strings.Repeat("-", barWidth)+"]") {
		t.Fatalf("zero-frac bar not fully empty, got:\n%s", out)
	}
}

func TestRenderResultTextCaveatsAndTakeawayOrder(t *testing.T) {
	var buf bytes.Buffer
	r := Result{Name: "x", Title: "X", Caveats: []string{"note one", "note two"}, Takeaway: "final word"}
	if err := RenderResultText(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	notesIdx := strings.Index(out, "note one")
	takeawayIdx := strings.Index(out, "Takeaway: final word")
	if notesIdx == -1 || takeawayIdx == -1 || notesIdx > takeawayIdx {
		t.Fatalf("caveats must render before the takeaway, got:\n%s", out)
	}
	if !strings.Contains(out, "note two") {
		t.Fatalf("second caveat missing, got:\n%s", out)
	}
}

func TestFormatPercentDecimals(t *testing.T) {
	if got := formatPercent(0.999, 0); got != "100%" {
		t.Fatalf("formatPercent(0.999, 0) = %q, want 100%%", got)
	}
	if got := formatPercent(0.999, 1); got != "99.9%" {
		t.Fatalf("formatPercent(0.999, 1) = %q, want 99.9%%", got)
	}
}

func TestShareOrDashZeroDenominator(t *testing.T) {
	if got := shareOrDash(5, 0, 1); got != "—" {
		t.Fatalf("shareOrDash(5, 0, 1) = %q, want —", got)
	}
}

func TestClamp01Bounds(t *testing.T) {
	cases := map[float64]float64{-1: 0, 0: 0, 0.5: 0.5, 1: 1, 2: 1}
	for in, want := range cases {
		if got := clamp01(in); got != want {
			t.Fatalf("clamp01(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestFracOfZeroMax(t *testing.T) {
	if got := fracOf(5, 0); got != 0 {
		t.Fatalf("fracOf(5, 0) = %v, want 0", got)
	}
}

func TestGroupLabelEmptyIsUnknown(t *testing.T) {
	if got := groupLabel(""); got != "(unknown)" {
		t.Fatalf("groupLabel(\"\") = %q, want (unknown)", got)
	}
}
