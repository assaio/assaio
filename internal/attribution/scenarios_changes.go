package attribution

import "time"

// changeScenarios hold pull requests a forge reports beside the repository: where a pull request's
// commit list can say which commits carry a session's work, and where it cannot say more than the
// commits already do.
var changeScenarios = []Scenario{
	{
		Name: "branch-commit-behind-a-squash",
		Defends: "the branch commit a session wrote is its evidence after a squash hid it from the main " +
			"line, and the squash belongs to the same pull request",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/feat.go"}, At: 30 * time.Minute},
			{Tag: "sq", Lands: "squash", From: "feat", Forge: true, At: hour + 5*time.Minute},
		},
		Sessions: []sessionSpec{work("s1")},
		Changes:  []changeSpec{{Tag: "P1", Number: 1, Lists: []string{"c1"}, Merge: "sq"}},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"c1"}, Change: "P1"}},
	},
	{
		Name: "two-authors-one-change",
		Defends: "two people's commits in one pull request place the session in that pull request and " +
			"still do not say which commit holds its work",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/one.go"}, At: hour + 5*time.Minute, Author: "Alice <alice@assaio.test>"},
			{Tag: "c2", Branch: "feat", Files: []string{"internal/app/two.go"}, At: hour + 6*time.Minute, Author: "Bob <bob@assaio.test>"},
			{Tag: "sq", Lands: "squash", From: "feat", Forge: true, At: 3 * day},
		},
		Sessions: []sessionSpec{work("s1")},
		Changes:  []changeSpec{{Tag: "P1", Number: 1, Lists: []string{"c1", "c2"}, Merge: "sq"}},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"c1", "c2"}, Ambiguous: true, Change: "P1"}},
	},
	{
		Name: "stacked-changes",
		Defends: "a commit two stacked pull requests both list belongs to neither alone; naming one is a " +
			"choice the evidence did not make",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Branch: "feat-a", Files: []string{"internal/app/a.go"}, At: hour + 5*time.Minute},
			{Tag: "sq", Lands: "squash", From: "feat-a", Forge: true, At: 3 * day},
		},
		Sessions: []sessionSpec{work("s1")},
		Changes: []changeSpec{
			{Tag: "A", Number: 1, Lists: []string{"c1"}, Merge: "sq"},
			{Tag: "B", Number: 2, Lists: []string{"c1", "c2-never-fetched"}, Updated: 4 * day},
		},
		Expect: map[string]Expectation{"s1": {Candidates: []string{"c1"}}},
	},
	{
		Name: "forge-update-branch-merge",
		Defends: "a merge the forge writes onto a pull request's branch carries no work, even when the " +
			"pull request lists it and this clone checked it out",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/feat.go"}, At: 30 * time.Minute},
			srcCommit("m1", 2*day),
			{Tag: "up", Lands: "merge", From: mainBranch, Onto: "feat", Forge: true, At: 3 * day},
		},
		Sessions: []sessionSpec{work("s1"), {ID: "s2", Start: 3*day - 30*time.Minute, End: 3*day + 30*time.Minute, Lines: 10}},
		Changes:  []changeSpec{{Tag: "P1", Number: 1, Lists: []string{"c1", "up"}, Updated: 3 * day}},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"c1"}, Change: "P1"}, "s2": {}},
	},
	{
		Name: "rebase-merge-two-commits",
		Defends: "a rebase merge replays every commit of a pull request, and each replayed commit belongs " +
			"to it, not only the last one the forge names",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/one.go"}, At: 30 * time.Minute},
			{Tag: "c2", Branch: "feat", Files: []string{"internal/app/two.go"}, At: 40 * time.Minute},
			{Tag: "rb1", Lands: "rebase", From: "c1", Forge: true, At: 3 * day},
			{Tag: "rb2", Lands: "rebase", From: "c2", Forge: true, At: 3*day + time.Minute},
		},
		Sessions: []sessionSpec{work("s1"), {ID: "s2", Start: 3*day - 30*time.Minute, End: 3*day + 30*time.Minute, Lines: 10}},
		Changes:  []changeSpec{{Tag: "P1", Number: 1, Lists: []string{"c1", "c2"}, Merge: "rb2"}},
		Expect: map[string]Expectation{
			"s1": {Candidates: []string{"c1", "c2", "rb1", "rb2"}, Change: "P1"},
			"s2": {},
		},
	},
	{
		Name: "one-candidate-outside-every-change",
		Defends: "a session whose candidates include a commit no pull request lists is not placed in the " +
			"pull request of the others",
		Commits: []commitSpec{
			srcCommit("base", -day),
			srcCommit("c1", 30*time.Minute),
			{Tag: "c9", Branch: "other", Files: []string{"internal/app/other.go"}, At: 50 * time.Minute},
		},
		Sessions: []sessionSpec{work("s1")},
		Changes:  []changeSpec{{Tag: "P1", Number: 1, Lists: []string{"c9"}, Updated: 2 * hour}},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"c1", "c9"}}},
	},
	{
		Name: "teammate-branch-commit-during-session",
		Defends: "a commit fetched from a teammate's branch is not this clone's work, even when a pull " +
			"request lists it and it was written during the session",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "theirs", Branch: "theirs", Fetched: true, At: 30 * time.Minute, Author: "Bob <bob@assaio.test>"},
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/feat.go"}, At: hour + 10*time.Minute},
		},
		Sessions: []sessionSpec{work("s1")},
		Changes: []changeSpec{
			{Tag: "P1", Number: 1, Lists: []string{"c1"}, Updated: 2 * hour},
			{Tag: "P2", Number: 2, Lists: []string{"theirs"}, Updated: 2 * hour},
		},
		Expect: map[string]Expectation{"s1": {Candidates: []string{"c1"}, Change: "P1"}},
	},
	{
		Name: "teammate-pull-request-checked-out-for-review",
		Defends: "a teammate's commit checked out to review it is still not this clone's work; a checkout " +
			"moves HEAD without making anything",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "theirs", Branch: "theirs", Fetched: true, CheckedOut: true, At: 30 * time.Minute, Author: "Bob <bob@assaio.test>"},
			{Tag: "c1", Branch: "feat", Files: []string{"internal/app/feat.go"}, At: hour + 10*time.Minute},
		},
		Sessions: []sessionSpec{work("s1")},
		Changes: []changeSpec{
			{Tag: "P1", Number: 1, Lists: []string{"c1"}, Updated: 2 * hour},
			{Tag: "P2", Number: 2, Lists: []string{"theirs"}, Updated: 2 * hour},
		},
		Expect: map[string]Expectation{"s1": {Candidates: []string{"c1"}, Change: "P1"}},
	},
	{
		Name: "adjacent-rebase-merges",
		Defends: "a rebase merge that wrote fewer commits than its pull request counts does not reach into " +
			"the pull request merged before it",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "a1", Branch: "feat-a", Fetched: true, At: 30 * time.Minute},
			{Tag: "ra1", Lands: "rebase", From: "a1", Forge: true, At: 2 * day},
			{Tag: "b1", Branch: "feat-b", Files: []string{"internal/app/b1.go"}, At: 3*day + 10*time.Minute},
			{Tag: "b2", Branch: "feat-b", Files: []string{"internal/app/b2.go"}, At: 3*day + 20*time.Minute},
			{Tag: "rb1", Lands: "rebase", From: "b1", Forge: true, At: 5 * day},
			{Tag: "rb2", Lands: "rebase", From: "b2", Forge: true, At: 5*day + time.Minute},
		},
		Sessions: []sessionSpec{work("s1"), {ID: "s2", Start: 3 * day, End: 3*day + hour, Lines: 10}},
		Changes: []changeSpec{
			{Tag: "A", Number: 1, Lists: []string{"a1"}, Merge: "ra1"},
			{Tag: "B", Number: 2, Lists: []string{"b1", "b2", "b3-dropped-by-the-replay"}, Merge: "rb2"},
		},
		Expect: map[string]Expectation{
			"s1": {Candidates: []string{"ra1"}, Change: "A"},
			"s2": {Candidates: []string{"b1", "b2", "rb1", "rb2"}, Change: "B"},
		},
	},
}
