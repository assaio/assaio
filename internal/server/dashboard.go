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
	in := analyze.BuildInput(usageRows, sessionRows, table, time.Now(), dashboardRecentWindow, analyze.Delegation{})
	const anonymize = true
	// Exec metric plugins are deliberately nil here: GET / is unauthenticated and
	// rebuilds per request, and spawning config-declared subprocesses per request would
	// be a denial-of-service vector. Compiled-in validators still run; exec metrics are
	// a local-CLI surface (ADR 0004).
	return dashboard.Build(in, "last 30 days", anonymize, nil, nil), nil
}
