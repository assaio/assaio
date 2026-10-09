package attribution

import "time"

func at(d time.Duration) *time.Duration { return &d }

// landingScenarios are the ways a change reaches the main line through a forge, and the local
// rewrites that separate the time a commit was written from the time it landed. The commit
// time git reports is the time of work only when the author committed straight onto the
// branch; each scenario here is a case where it is not.
var landingScenarios = []Scenario{
	{
		Name: "squash-landed-after-the-session",
		Defends: "a squash the forge lands minutes after the session carries that session's work; " +
			"stamped at the merge, it is still a candidate",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/feat.go"}, At: 30 * time.Minute},
			{Tag: "sq", Lands: "squash", From: "feat", Forge: true, At: hour + 5*time.Minute},
		},
		Sessions: []sessionSpec{work("s1")},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"sq"}}},
	},
	{
		Name: "squash-landed-in-another-session",
		Defends: "a squash stamped while another session was running is as plausibly the earlier " +
			"session's work as that one's; neither may be dropped",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/feat.go"}, At: 30 * time.Minute},
			{Tag: "sq", Lands: "squash", From: "feat", Forge: true, At: 25*hour + 30*time.Minute},
		},
		Sessions: []sessionSpec{work("s1"), {ID: "s2", Start: 25 * hour, End: 26 * hour, Lines: 10}},
		Expect: map[string]Expectation{
			"s1": {Candidates: []string{"sq"}},
			"s2": {Candidates: []string{"sq"}},
		},
	},
	{
		Name: "rebase-merge",
		Defends: "a forge that replays a commit keeps the time it was written; the session that wrote " +
			"it keeps the link and the one active at the merge does not take it",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/feat.go"}, At: 30 * time.Minute},
			{Tag: "rb", Lands: "rebase", From: "c1", Forge: true, At: 3 * day},
		},
		Sessions: []sessionSpec{work("s1"), {ID: "s2", Start: 3*day - 30*time.Minute, End: 3*day + 30*time.Minute, Lines: 10}},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"rb"}}, "s2": {}},
	},
	{
		Name: "forge-merge-commit",
		Defends: "a merge the forge writes carries no work of its own; the branch commits it brings " +
			"in keep their sessions and the session active at the merge gets nothing from it",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/feat.go"}, At: 30 * time.Minute},
			{Tag: "mg", Lands: "merge", From: "feat", Forge: true, At: 3 * day},
		},
		Sessions: []sessionSpec{work("s1"), {ID: "s2", Start: 3*day - 30*time.Minute, End: 3*day + 30*time.Minute, Lines: 10}},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"c1"}}, "s2": {}},
	},
	{
		Name: "amended-in-a-later-session",
		Defends: "a commit written in one session and amended in another carries both sessions' work; " +
			"judging it by either of its two times alone drops one of them",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Files: []string{"internal/app/one.go"}, At: 3*day + 10*time.Minute, Authored: at(30 * time.Minute)},
		},
		Sessions: []sessionSpec{work("s1"), {ID: "s2", Start: 3 * day, End: 3*day + hour, Lines: 10}},
		Expect: map[string]Expectation{
			"s1": {Candidates: []string{"c1"}},
			"s2": {Candidates: []string{"c1"}},
		},
	},
	{
		Name: "local-merge-carries-work",
		Defends: "a merge made locally can carry a conflict resolution, and git reports no lines for any " +
			"merge; only the forge's own merges are known to carry nothing",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c-side", Branch: "side", Files: []string{"internal/app/side.go"}, At: -12 * hour},
			{Tag: "mg", Lands: "merge", From: "side", At: 40 * time.Minute},
		},
		Sessions: []sessionSpec{work("s1")},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"mg"}}},
	},
	{
		Name:    "confirmed-forge-merge-survives",
		Defends: "a human's confirmation outranks the rule that a forge merge carries no work",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/feat.go"}, At: 2 * day},
			{Tag: "mg", Lands: "merge", From: "feat", Forge: true, At: 3 * day},
		},
		Sessions:    []sessionSpec{{ID: "s1", Start: 3*day - 30*time.Minute, End: 3*day + 30*time.Minute, Lines: 10}},
		Corrections: map[string]string{"s1": "mg"},
		Expect:      map[string]Expectation{"s1": {Candidates: []string{"mg"}, Confirmed: "mg"}},
	},
}
