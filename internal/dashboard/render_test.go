package dashboard

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

var update = flag.Bool("update", false, "update golden files")

// fixedPseudonymKey seeds report.Pseudonym's per-install secret so this package's
// anonymized output -- including the golden HTML files, which embed pseudonymized
// labels -- is byte-reproducible across machines and runs, rather than depending on a
// randomly generated key.
var fixedPseudonymKey = bytes.Repeat([]byte{0x5a}, 32)

// TestMain gives every test in this package a hermetic, fixed data directory. Dashboard
// tests call Build/RenderHTML directly rather than through the CLI's own per-test
// XDG_DATA_HOME convention, and report.Pseudonym now persists a per-install secret to
// disk (see internal/report/anonymize.go) -- without this, tests would read and write the
// real user's data directory, and golden-file comparisons would depend on whatever key
// happened to already be there.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "assaio-dashboard-test")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_DATA_HOME", dir); err != nil {
		panic(err)
	}
	dataDir, err := paths.DataDir()
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		panic(err)
	}
	// The filename must match internal/report/anonymize.go's own pseudonymKeyFilename;
	// duplicated here rather than exported since it is a test-only seam.
	keyPath := filepath.Join(dataDir, "pseudonym.key")
	if err := os.WriteFile(keyPath, fixedPseudonymKey, 0o600); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// TestPseudonymKeyIsSeededDeterministically guards the TestMain seam above: if
// report.Pseudonym's key file name or size ever changes, this fails loudly instead of
// this package's golden files silently starting to compare against a random key.
func TestPseudonymKeyIsSeededDeterministically(t *testing.T) {
	got := report.Pseudonym("project", "acme-web")
	want := report.Pseudonym("project", "acme-web")
	if got != want {
		t.Fatalf("Pseudonym must be stable within a test run: %q != %q", got, want)
	}
}

func TestRenderHTMLGoldenAnonymized(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", true, fixtureSubpaths(), nil)
	renderAndCompareGolden(t, &d, "testdata/dashboard.golden.html")
}

func TestRenderHTMLGoldenRealNames(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", false, fixtureSubpaths(), nil)
	renderAndCompareGolden(t, &d, "testdata/dashboard_real.golden.html")
}

func renderAndCompareGolden(t *testing.T, d *Data, golden string) {
	t.Helper()
	var buf bytes.Buffer
	if err := RenderHTML(&buf, *d); err != nil {
		t.Fatal(err)
	}
	got := buf.Bytes()
	if *update {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden) //nolint:gosec // golden is a fixed testdata path, not user input
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch for %s:\n got=%s\nwant=%s", golden, got, want)
	}
}

// TestRenderHTMLSelfContained enforces the hard "no network requests" rule: the report
// is opened offline from disk, so besides its one inline theme-toggle script it must
// carry no external references at all -- no CDN scripts/fonts, no remote images, and no
// externally sourced script.
func TestRenderHTMLSelfContained(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderHTML(&buf, Build(fixtureInput(), "last 30 days", true, fixtureSubpaths(), nil)); err != nil {
		t.Fatal(err)
	}
	html := strings.ToLower(buf.String())
	for _, needle := range []string{"http://", "https://", "src=", "cdn", "<script src"} {
		if strings.Contains(html, needle) {
			t.Fatalf("dashboard HTML must be self-contained: found forbidden %q", needle)
		}
	}
}

func TestRenderHTMLContainsExpectedContent(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderHTML(&buf, Build(fixtureInput(), "last 30 days", true, fixtureSubpaths(), nil)); err != nil {
		t.Fatal(err)
	}
	html := strings.ToUpper(buf.String())
	for _, want := range []string{
		"ASSAIO", "ASSAY VERDICT", "ALL PROJECTS", "ADOPTION", "MODEL FIT", "CONTEXT", "THROUGHPUT", "REWORK",
		"DIRECTIONAL ASSAY", "AGGREGATE AND PSEUDONYMIZED BY DEFAULT", "SUBPATH BREAKDOWN", "COST BASIS",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("dashboard HTML missing %q", want)
		}
	}
	if !strings.Contains(buf.String(), "project-") {
		t.Fatal("anonymized dashboard must show at least one pseudonymized project label")
	}
}

// TestRenderHTMLRealNamesShowActualProjects checks for the --no-anonymize pointer
// (unique to the per-render AnonymizedCaveat) rather than the bare word "pseudonymized":
// the always-present DirectionalCaveat now legitimately mentions pseudonymization as
// assaio's default posture, independent of whether this render used real names.
func TestRenderHTMLRealNamesShowActualProjects(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderHTML(&buf, Build(fixtureInput(), "last 30 days", false, fixtureSubpaths(), nil)); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if !strings.Contains(html, ">api<") {
		t.Fatalf("real-name dashboard must show the actual top project name: %s", html)
	}
	if strings.Contains(html, "--no-anonymize") {
		t.Fatal("real-name dashboard must not carry the per-render pseudonymization caveat")
	}
}

// TestRenderHTMLEmptyInputIsValidHonestPage asserts an empty store still renders a valid
// page -- every validator's honest no-data takeaway, and no drill-down section at all
// (there is no project to drill into), rather than a crash or a misleading blank result.
func TestRenderHTMLEmptyInputIsValidHonestPage(t *testing.T) {
	in := analyze.BuildInput(nil, nil, fixturePrices(), fixtureNow, 7*24*time.Hour, analyze.Delegation{})
	var buf bytes.Buffer
	if err := RenderHTML(&buf, Build(in, "last 30 days", true, nil, nil)); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if !strings.Contains(html, "No usage in this window.") {
		t.Fatalf("empty dashboard must show every validator's honest no-data takeaway: %s", html)
	}
	if strings.Contains(html, `class="drilldown"`) {
		t.Fatal("empty dashboard must omit the drill-down section entirely (no project to drill into)")
	}
	if got := strings.Count(html, `class="entry__howto"`); got != len(analyze.Validators()) {
		t.Fatalf("entry__howto count = %d on empty input, want one per validator (%d): HowToRead must render even with no data", got, len(analyze.Validators()))
	}
}

// TestRenderHTMLIncludesHowToReadLines asserts the ledger shows one muted "how to read"
// helper line per validator, sourced from analyze.Result.HowToRead -- the same text the
// CLI's "? " line renders (see analyze.RenderResultText).
func TestRenderHTMLIncludesHowToReadLines(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderHTML(&buf, Build(fixtureInput(), "last 30 days", true, fixtureSubpaths(), nil)); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if got := strings.Count(html, `class="entry__howto"`); got != len(analyze.Validators()) {
		t.Fatalf("entry__howto count = %d, want one per validator (%d)", got, len(analyze.Validators()))
	}
	if !strings.Contains(html, "a place to trim spend without losing output") {
		t.Fatalf("model-fit's HowToRead text missing from the dashboard: %s", html)
	}
}

// TestRenderHTMLFooterDropsAbsoluteNeverPerPerson locks in the Improvement 3 reframe:
// the colophon must no longer claim an absolute "never per person" and must instead
// state the aggregate/pseudonymized-by-default posture with its team-mode opt-in.
func TestRenderHTMLFooterDropsAbsoluteNeverPerPerson(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderHTML(&buf, Build(fixtureInput(), "last 30 days", true, fixtureSubpaths(), nil)); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if strings.Contains(strings.ToLower(html), "never per person") {
		t.Fatalf("footer must not claim an absolute never-per-person stance: %s", html)
	}
	if !strings.Contains(html, "per-person requires an explicit opt-in (team mode)") {
		t.Fatalf("footer must state per-person requires an explicit opt-in: %s", html)
	}
}

// TestRenderHTMLEscapesUntrustedNames is the finding-4 regression: project, model,
// member, and subpath names come straight from local session logs -- untrusted input --
// and the offline HTML report must never let one of them break out of its text node.
// html/template auto-escapes by default and the template never uses template.HTML, so
// this is already safe; this test locks that in with real script/attribute-breakout
// payloads rather than leaving it unverified.
func TestRenderHTMLEscapesUntrustedNames(t *testing.T) {
	const (
		xssProject = `<script>alert('assay-xss-project')</script>`
		xssModel   = `"><img src=x onerror=alert('assay-xss-model')>`
		xssMember  = `<img src=x onerror=alert('assay-xss-member')>`
		xssSubpath = `<script>alert('assay-xss-subpath')</script>`
	)
	escaped := map[string]string{
		xssProject: `&lt;script&gt;alert(&#39;assay-xss-project&#39;)&lt;/script&gt;`,
		xssModel:   `&#34;&gt;&lt;img src=x onerror=alert(&#39;assay-xss-model&#39;)&gt;`,
		xssMember:  `&lt;img src=x onerror=alert(&#39;assay-xss-member&#39;)&gt;`,
		xssSubpath: `&lt;script&gt;alert(&#39;assay-xss-subpath&#39;)&lt;/script&gt;`,
	}

	usage := []store.UsageRow{
		{
			Day: fixtureNow.Format("2006-01-02"), Tool: "claude-code", Model: xssModel, Project: xssProject,
			Member: xssMember, In: 1000, Out: 2000, LinesAdded: 500, Edits: 2, ToolCalls: 3,
		},
	}
	sessions := []store.SessionRow{
		{
			SessionID: "s1", Project: xssProject, Tool: "claude-code", Model: xssModel, Member: xssMember,
			FirstTs: fixtureNow.Add(-time.Hour), LastTs: fixtureNow, Turns: 2, OutputTokens: 2000, Edits: 2, ActiveMinutes: 10,
		},
	}
	in := analyze.BuildInput(usage, sessions, fixturePrices(), fixtureNow, 7*24*time.Hour, analyze.Delegation{})
	subpaths := []store.SubpathRow{{Subpath: xssSubpath, Lines: 500, Sessions: 1}}

	// anonymize=false: the real, attacker-controlled strings must reach the template
	// unmodified, so this test actually exercises escaping rather than pseudonymization.
	d := Build(in, "last 30 days", false, subpaths, nil)
	if d.Drill == nil {
		t.Fatal("fixture must produce a top project to drill into -- the payloads flow through Drill.Name and the subpath table")
	}
	var buf bytes.Buffer
	if err := RenderHTML(&buf, d); err != nil {
		t.Fatal(err)
	}
	html := buf.String()

	for payload, want := range escaped {
		if strings.Contains(html, payload) {
			t.Fatalf("dashboard HTML must escape untrusted input, found raw payload %q in output", payload)
		}
		if !strings.Contains(html, want) {
			t.Fatalf("dashboard HTML must contain the escaped form of %q (want %q)", payload, want)
		}
	}
}

func TestClampPurity(t *testing.T) {
	tests := []struct{ in, want float64 }{
		{-3, 0},
		{-0.0001, 0},
		{0, 0},
		{0.5, 0.5},
		{1, 1},
		{1.0001, 1},
		{5, 1},
	}
	for _, tt := range tests {
		if got := clampPurity(tt.in); got != tt.want {
			t.Fatalf("clampPurity(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestFacetsClampsPurity(t *testing.T) {
	in := []analyze.Result{{Purity: 5}, {Purity: -3}, {Purity: 0.4}}
	out := facets(in)
	want := []float64{1, 0, 0.4}
	for i, w := range want {
		if out[i].R.Purity != w {
			t.Fatalf("facets()[%d].R.Purity = %v, want %v", i, out[i].R.Purity, w)
		}
	}
}

func TestRenderHTMLClampsOutOfRangePurityGauge(t *testing.T) {
	tooHigh := analyze.Result{
		Name: "custom-high", Title: "Custom High", Read: analyze.Read{Key: "good", Label: "OK"},
		Purity: 5, Takeaway: "t",
	}
	tooLow := analyze.Result{
		Name: "custom-low", Title: "Custom Low", Read: analyze.Read{Key: "watch", Label: "WATCH"},
		Purity: -3, Takeaway: "t",
	}
	d := Data{Window: "last 30 days", Verdicts: []analyze.Result{tooHigh, tooLow}, CostBasis: "—"}
	var buf bytes.Buffer
	if err := RenderHTML(&buf, d); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if strings.Contains(html, "--fill:500.0%") {
		t.Fatalf("Purity > 1 must be clamped, not rendered as an overflowing gauge width: %s", html)
	}
	if !strings.Contains(html, "--fill:100.0%") {
		t.Fatalf("Purity > 1 must clamp the gauge fill to 100%%: %s", html)
	}
	if strings.Contains(html, "--fill:-300.0%") {
		t.Fatalf("Purity < 0 must be clamped, not rendered as a negative gauge width: %s", html)
	}
	if !strings.Contains(html, "--fill:0.0%") {
		t.Fatalf("Purity < 0 must clamp the gauge fill to 0%%: %s", html)
	}
}
