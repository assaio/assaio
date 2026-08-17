package share

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
)

// secret is planted into every user-chosen string a row carries, not just the project. The
// first version of this test planted only Project, which is why nothing caught that an
// exec-plugin tool is stored as "plugin:<name>" where the name comes from the user's own
// config -- a label that reached the card and is not a vendor fact.
const secret = "acme-internal-billing"

func plantedInput(t *testing.T) analyze.Input {
	t.Helper()
	in := sampleInput(t)
	for i := range in.Usage {
		in.Usage[i].Project = secret
		in.Usage[i].Member = secret
		in.Usage[i].Entrypoint = secret
		if i%3 == 0 {
			in.Usage[i].Tool = "plugin:" + secret
		}
	}
	for i := range in.Sessions {
		in.Sessions[i].Project = secret
		in.Sessions[i].Member = secret
	}
	return in
}

// TestNoUserChosenNameReachesAnySurface renders every output the command can produce and
// fails if any of them carries a string the user picked. It walks the marshalled payload as
// well as the rendered text, because the preview page ships the whole Assay as JSON and a
// field nothing draws today is still a field that left the machine.
func TestNoUserChosenNameReachesAnySurface(t *testing.T) {
	in := plantedInput(t)
	results := make([]analyze.Result, 0)
	for _, v := range analyze.Validators() {
		results = append(results, analyze.Evaluate(v, &in))
	}
	a := Build(in, results, "last 30 days", false)

	var html bytes.Buffer
	if err := RenderHTML(&html, &a); err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	payload, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for name, body := range map[string]string{
		"html":    html.String(),
		"text":    Text(&a),
		"post":    a.Post,
		"payload": string(payload),
	} {
		if strings.Contains(body, secret) {
			t.Errorf("%s carries the user-chosen name %q", name, secret)
		}
	}
}

// TestPluginToolIsNotNamed pins the rule directly, so a future change to toolSlices cannot
// quietly restore the label while the broader test above still passes for another reason.
func TestPluginToolIsNotNamed(t *testing.T) {
	tests := []struct {
		tool string
		want string
	}{
		{"claude-code", "claude-code"},
		{"codex", "codex"},
		{"cline", "cline"},
		{"plugin:" + secret, "a plugin source"},
		{"something-nobody-registered", "a plugin source"},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			if got := publishableTool(tt.tool); got != tt.want {
				t.Errorf("publishableTool(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

// TestEmptyWindowRendersWithoutNulls covers the shape that killed the preview: a nil slice
// marshals to `null`, the page walks four of them unguarded, and the throw ended the draw
// loop -- leaving the canvas on its background, which is what "Save the reel" would then
// have recorded.
func TestEmptyWindowRendersWithoutNulls(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	in := analyze.BuildInput(nil, nil, nil, now, 7*24*time.Hour, analyze.Delegation{})
	results := make([]analyze.Result, 0)
	for _, v := range analyze.Validators() {
		results = append(results, analyze.Evaluate(v, &in))
	}
	a := Build(in, results, "last 30 days", false)

	payload, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, field := range []string{`"Models":null`, `"Tools":null`, `"Rays":null`, `"Axes":null`, `"Ledger":null`, `"Caveats":null`} {
		if bytes.Contains(payload, []byte(field)) {
			t.Errorf("payload carries %s; the renderer walks it unguarded", field)
		}
	}
	if a.Hook.Line == "" {
		t.Error("an empty window still needs a first frame")
	}
	if a.Limit == "" {
		t.Error("an empty window still needs its limit line")
	}
	var html bytes.Buffer
	if err := RenderHTML(&html, &a); err != nil {
		t.Fatalf("RenderHTML on an empty window: %v", err)
	}
}

// TestReserveNamesTheSideTheDataIsOn is the defect the earlier assertion was too loose to
// see: the fallback named Axis.Right unconditionally while the widest lean is as often to
// the left, so a window with little rework was published as "reworks".
func TestReserveNamesTheSideTheDataIsOn(t *testing.T) {
	tests := []struct {
		name    string
		facts   facts
		wantNot string
	}{
		{"little rework leans to first try", facts{rework: num{v: 4, ok: true, s: "4%"}}, "reworks"},
		{"thrifty leans away from premium", facts{premium: num{v: 8, ok: true, s: "8%"}}, "premium"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.facts
			p := buildProfile(&f)
			if p.Line == "" {
				t.Fatal("no reserve line")
			}
			if strings.HasPrefix(p.Line, tt.wantNot+" ") {
				t.Errorf("reserve line %q names the pole the data leans away from", p.Line)
			}
		})
	}
}
