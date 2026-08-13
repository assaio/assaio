package server

import (
	"context"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/dashboard"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// DashboardBuilder builds the Assay dashboard's render-ready Data from the central
// store's current usage. Injected into New so a caller (or a test) can substitute a
// stub without this package depending on how that builder is implemented.
type DashboardBuilder func(ctx context.Context, st *store.Store) (dashboard.Data, error)

// dashboardWindow bounds how far back GET / looks; the MVP has no query-string override.
const dashboardWindow = 30 * 24 * time.Hour

// dashboardRecentWindow is the recent-vs-prior window validators use for trend and
// staleness signals, matching the CLI dashboard's own window (internal/cli/analyze.go).
const dashboardRecentWindow = 7 * 24 * time.Hour

// BuildDashboard is the default DashboardBuilder: the whole team's usage across every
// member, anonymized by default -- aggregate and pseudonymized is assaio's default
// privacy mode (AGENTS.md). dashboard.Build adds a Team section automatically whenever
// the store carries member data, with member labels pseudonymized by this hardcoded
// anonymize=true; showing real member names is a CLI-only, explicit opt-in
// (`--no-anonymize` against this same store via internal/cli's --db flag), never this
// served endpoint's default.
func BuildDashboard(ctx context.Context, st *store.Store) (dashboard.Data, error) {
	since := time.Now().Add(-dashboardWindow)
	usageRows, err := st.Usage(ctx, since)
	if err != nil {
		return dashboard.Data{}, err
	}
	sessionRows, err := st.Sessions(ctx, since)
	if err != nil {
		return dashboard.Data{}, err
	}
	table, err := pricing.Load()
	if err != nil {
		return dashboard.Data{}, err
	}
	turns, err := st.TurnSizing(ctx, since, analyze.RightSizeSmallOutput)
	if err != nil {
		return dashboard.Data{}, err
	}
	sub, total, err := st.Delegation(ctx, since)
	if err != nil {
		return dashboard.Data{}, err
	}
	skills, agents, err := st.Attribution(ctx, since)
	if err != nil {
		return dashboard.Data{}, err
	}
	misses, err := st.CacheMisses(ctx, since)
	if err != nil {
		return dashboard.Data{}, err
	}
	in := analyze.BuildInput(usageRows, sessionRows, table, time.Now(), dashboardRecentWindow, analyze.Delegation{Sub: sub, Total: total})
	in.WindowStart = since
	in.TurnSizing = turns
	in.Skills, in.Agents = skills, agents
	in.CacheMisses = misses
	in.Ingested, in.ParsedBy, _ = st.Provenance(ctx)
	// Both are read here for the same reason the CLI reads them: a trending panel that cannot see
	// how far back the store goes disclaims its own figure, and a detector with no sequences says
	// so rather than reporting a zero. A synced store holds no sequences by construction (ADR
	// 0012), which is a fact for the panel to state, not a reason to leave the field unset.
	if oldest, histErr := st.HistoryStart(ctx, ""); histErr == nil {
		in.HistoryStart = oldest
	}
	// The sequences are deliberately NOT read here, for the reason the metric plugins are not: this
	// handler is unauthenticated and rebuilds on every request with no cache, and Timelines is a
	// full scan of the step table plus a GROUP BY over the window's records -- about 2.5s on a
	// 339,000-step store, run before it can know whether any step exists. A store filled by `sync`
	// holds none at all, since the team-server contract carries usage records and not sequences
	// (ADR 0012), so the read would cost that on every request to return nothing. The detectors
	// therefore report no sequences here and say why, which is true of this surface (`B171`).
	const anonymize = true
	// Exec metric plugins are deliberately nil here: GET / is unauthenticated and
	// rebuilds per request, and spawning config-declared subprocesses per request would
	// be a denial-of-service vector. Compiled-in validators still run; exec metrics are
	// a local-CLI surface (ADR 0004).
	return dashboard.Build(in, "last 30 days", anonymize, nil, nil), nil
}
