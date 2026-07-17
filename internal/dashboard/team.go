package dashboard

import (
	"sort"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// TeamStat is one member's adoption signal for the dashboard's ranked bar list: how much
// they engaged, never how "good" their output was. Frac is Sessions scaled against the
// list's own maximum, for the bar width.
type TeamStat struct {
	Member      string
	Sessions    int
	LinesAdded  int64
	CostDisplay string
	Frac        float64
}

// Team is the dashboard's per-member adoption breakdown: present only when the queried
// usage carries a non-empty Member, i.e. it was aggregated by a team server from synced
// agents -- never for a purely local store (see buildTeam).
type Team struct {
	Stats []TeamStat
}

// buildTeam returns nil when usage carries no member data at all, so a purely local
// store's Data.Team stays absent rather than an empty section. Stats are ranked by
// session count -- engagement frequency, an adoption-spread signal -- deliberately never
// by cost or lines added, which would read as a productivity scoreboard (AGENTS.md
// honesty rules: aggregate/pseudonymized by default, never a leaderboard). Member labels
// are pseudonymized by default via report.Pseudonym, matching project names' own
// opt-in-to-reveal convention.
func buildTeam(usageRows []store.UsageRow, sessionRows []store.SessionRow, prices pricing.Table, anonymize bool) *Team {
	if !hasMemberData(usageRows) {
		return nil
	}

	lines, cost, hasCost := memberTotals(usageRows, prices)
	sessions, members := memberSessionCounts(sessionRows)
	for m := range lines {
		members[m] = struct{}{}
	}

	names := rankedMemberNames(members, sessions)
	maxSessions := 0
	for _, m := range names {
		if sessions[m] > maxSessions {
			maxSessions = sessions[m]
		}
	}

	stats := make([]TeamStat, 0, len(names))
	for _, m := range names {
		stats = append(stats, TeamStat{
			Member:      memberLabel(m, anonymize),
			Sessions:    sessions[m],
			LinesAdded:  lines[m],
			CostDisplay: costDisplay(cost[m], hasCost[m]),
			Frac:        fraction(sessions[m], maxSessions),
		})
	}
	return &Team{Stats: stats}
}

// hasMemberData reports whether any row carries a non-empty Member -- the signal that
// usage came from a central store synced from at least one team member, not a purely
// local install.
func hasMemberData(rows []store.UsageRow) bool {
	for i := range rows {
		if rows[i].Member != "" {
			return true
		}
	}
	return false
}

// memberTotals sums each member's AI lines and (when priced) cost across usageRows.
func memberTotals(rows []store.UsageRow, prices pricing.Table) (lines map[string]int64, cost map[string]float64, hasCost map[string]bool) {
	lines = make(map[string]int64)
	cost = make(map[string]float64)
	hasCost = make(map[string]bool)
	for i := range rows {
		m := rows[i].Member
		lines[m] += rows[i].LinesAdded
		if c, ok := prices.CostTokens(rows[i].Model, rows[i].In, rows[i].Out, rows[i].CacheWrite, rows[i].CacheRead); ok {
			cost[m] += c
			hasCost[m] = true
		}
	}
	return lines, cost, hasCost
}

// memberSessionCounts counts sessions per member and returns the set of members seen.
func memberSessionCounts(rows []store.SessionRow) (counts map[string]int, members map[string]struct{}) {
	counts = make(map[string]int)
	members = make(map[string]struct{})
	for i := range rows {
		m := rows[i].Member
		counts[m]++
		members[m] = struct{}{}
	}
	return counts, members
}

// rankedMemberNames orders members by session count descending -- adoption spread, not a
// performance ranking -- breaking ties alphabetically for deterministic output.
func rankedMemberNames(members map[string]struct{}, sessions map[string]int) []string {
	names := make([]string, 0, len(members))
	for m := range members {
		names = append(names, m)
	}
	sort.Slice(names, func(i, j int) bool {
		if sessions[names[i]] != sessions[names[j]] {
			return sessions[names[i]] > sessions[names[j]]
		}
		return names[i] < names[j]
	})
	return names
}

// memberLabel renders m for display: "(local)" for usage with no member attribution
// (pre-sync or server-local records), else the real name or, by default, its pseudonym.
func memberLabel(m string, anonymize bool) string {
	if m == "" {
		return "(local)"
	}
	if anonymize {
		return report.Pseudonym("member", m)
	}
	return m
}

// costDisplay renders a member's summed cost compactly, or "—" when none of their usage
// was priced -- never a fabricated zero.
func costDisplay(cost float64, hasCost bool) string {
	if !hasCost {
		return "—"
	}
	return formatCompactUSD(cost)
}

// fraction is v/max, 0 when max is 0 -- never a divide-by-zero.
func fraction(v, max int) float64 {
	if max == 0 {
		return 0
	}
	return float64(v) / float64(max)
}
