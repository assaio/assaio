package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/trace"
)

// TestEnvelopeSizeOnARealStore is how the figures in ADR 0004 were produced: the same envelope
// builder the runtime uses, over a real window, under the projection each kind of plugin
// declares. It is skipped unless ASSAIO_MEASURE_DB names a store, because the only honest
// version of this measurement is over rows somebody actually generated -- a synthetic window
// would size a fixture rather than the protocol.
//
// The named file is copied before it is opened. store.Open migrates, so pointing a test at a
// store is a write, and the store this is worth running against is the one nobody can rebuild.
func TestEnvelopeSizeOnARealStore(t *testing.T) {
	path := os.Getenv("ASSAIO_MEASURE_DB")
	if path == "" {
		t.Skip("set ASSAIO_MEASURE_DB to a store to size the envelope against it")
	}
	st, err := store.Open(copyOfStore(t, path))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	days := 30
	if v := os.Getenv("ASSAIO_MEASURE_DAYS"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &days)
	}
	start := time.Now().AddDate(0, 0, -days)

	usage, err := st.UsageFiltered(ctx, start, store.LabelFilter{})
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := st.SessionsFiltered(ctx, start, store.LabelFilter{})
	if err != nil {
		t.Fatal(err)
	}
	sub, total, err := st.DelegationFiltered(ctx, start, store.LabelFilter{})
	if err != nil {
		t.Fatal(err)
	}
	table, err := pricing.Load()
	if err != nil {
		t.Fatal(err)
	}
	in := analyze.BuildInput(usage, sessions, table, time.Now(), 7*24*time.Hour, analyze.Delegation{Sub: sub, Total: total})
	in.WindowStart = start
	in.TurnSizing, _ = st.TurnSizingFiltered(ctx, start, analyze.RightSizeSmallOutput, store.LabelFilter{})
	in.Skills, in.Agents, _ = st.AttributionFiltered(ctx, start, store.LabelFilter{})
	in.CacheMisses, _ = st.CacheMissesFiltered(ctx, start, store.LabelFilter{})
	sequences, err := st.Timelines(ctx, start)
	if err != nil {
		t.Fatal(err)
	}
	set := trace.New(sequences)
	in.Trace = set.ForSessions(sessions)
	in.HistoryStart, _ = st.HistoryStart(ctx, "")

	steps := 0
	for _, s := range in.Trace.All() {
		steps += len(s.Steps)
	}
	t.Logf("window: %d days, %d usage rows, %d sessions, %d sequences, %d steps",
		days, len(in.Usage), len(in.Sessions), len(in.Trace.All()), steps)

	cheap := []analyze.Capability{
		analyze.CapUsage, analyze.CapSessions, analyze.CapAttribution,
		analyze.CapTurnSizing, analyze.CapCacheMisses, analyze.CapPrices,
	}
	cases := []struct {
		what string
		p    Projection
	}{
		{"protocol 3, no `needs:` (everything but trace)", Projection{Needs: cheap}},
		{"protocol 3, `needs: [trace]` (everything)", Projection{Needs: analyze.Capabilities()}},
		{"protocol 4, a token-share metric: needs [usage], 4 columns", Projection{
			Needs:  []analyze.Capability{analyze.CapUsage},
			Fields: map[string][]string{"usage": {"day", "tool", "in", "out"}},
		}},
		{"protocol 4, a step detector: needs [trace], 2 step columns, interactive only", Projection{
			Needs: []analyze.Capability{analyze.CapTrace},
			Fields: map[string][]string{
				"trace": {"sessionId", "scope", "steps"}, "trace.steps": {"kind", "outcome"},
			},
			Where: map[string][]string{"trace.scope": {"interactive"}},
		}},
	}
	for _, c := range cases {
		envelope := buildMetricInput(&in, c.p)
		out, err := envelope.marshal()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%-62s %12d bytes  (%6.2f MB)", c.what, len(out), float64(len(out))/(1<<20))
	}
}

func copyOfStore(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // the operator names the store on the command line
	if err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(t.TempDir(), "measured.db")
	//nolint:gosec // copied is a path under this test's own TempDir
	if err := os.WriteFile(copied, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return copied
}
