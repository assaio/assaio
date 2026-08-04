package attribution

import "time"

const (
	hour = time.Hour
	day  = 24 * time.Hour
)

// work is the session shape most scenarios reuse: a morning of work under one id.
func work(id string) sessionSpec {
	return sessionSpec{ID: id, Start: 0, End: hour, Lines: 120}
}

// srcCommit is the commit shape most scenarios reuse: one source file, on the main line.
func srcCommit(tag string, at time.Duration) commitSpec {
	return commitSpec{Tag: tag, Files: []string{"internal/app/" + tag + ".go"}, At: at}
}

// scenarios is the corpus, ordered as it is worth reading: the shapes a link takes, the
// noise an engine has to resist, and the cases where it must refuse to answer.
var scenarios = []Scenario{
	{
		Name:     "one-session-one-commit",
		Defends:  "the simple case stays simple: one session, one commit, one link",
		Commits:  []commitSpec{srcCommit("c1", hour+5*time.Minute)},
		Sessions: []sessionSpec{work("s1")},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"c1"}}},
	},
	{
		Name:     "many-sessions-one-commit",
		Defends:  "two sessions can produce one commit; neither may be dropped to make the link one-to-one",
		Commits:  []commitSpec{srcCommit("c1", hour+5*time.Minute)},
		Sessions: []sessionSpec{work("s1"), work("s2")},
		Expect: map[string]Expectation{
			"s1": {Candidates: []string{"c1"}},
			"s2": {Candidates: []string{"c1"}},
		},
	},
	{
		Name:     "one-session-many-commits",
		Defends:  "one session can produce several commits; keeping only the closest loses the rest of its work",
		Commits:  []commitSpec{srcCommit("c1", hour+5*time.Minute), srcCommit("c2", 2*hour)},
		Sessions: []sessionSpec{{ID: "s1", Start: 0, End: 3 * hour, Lines: 300}},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"c1", "c2"}}},
	},
	{
		Name: "wrong-branch",
		Defends: "a commit on a branch that never reached the main line is not evidence, " +
			"and a session with no evidence stays unattributed",
		Commits: []commitSpec{
			srcCommit("base", -day),
			{Tag: "c-side", Branch: "side", Files: []string{"internal/app/side.go"}, At: hour + 5*time.Minute},
		},
		Sessions: []sessionSpec{work("s1")},
		Expect:   map[string]Expectation{"s1": {}},
	},
	{
		Name: "delayed-commit",
		Defends: "work committed a day later is still that session's work; a window narrow enough " +
			"to look precise silently drops it",
		Commits:  []commitSpec{srcCommit("c1", day+2*hour)},
		Sessions: []sessionSpec{work("s1")},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"c1"}}},
	},
	{
		Name: "sub-agent",
		Defends: "a sub-agent's tokens are part of its parent's work, so both sessions reach the " +
			"same commit rather than the child being orphaned",
		Commits: []commitSpec{srcCommit("c1", hour+5*time.Minute)},
		Sessions: []sessionSpec{
			work("s1"),
			{ID: "s1-sub", Parent: "s1", Start: 10 * time.Minute, End: 40 * time.Minute, Lines: 60},
		},
		Expect: map[string]Expectation{
			"s1":     {Candidates: []string{"c1"}},
			"s1-sub": {Candidates: []string{"c1"}},
		},
	},
	{
		Name: "overlapping-users",
		Defends: "a commit observation carries no author, so two people working the same window " +
			"cannot be separated by identity and the ambiguity has to be reported",
		Commits: []commitSpec{
			{Tag: "c1", Files: []string{"internal/app/one.go"}, At: hour + 5*time.Minute, Author: "Alice <alice@assaio.test>"},
			{Tag: "c2", Files: []string{"internal/app/two.go"}, At: hour + 6*time.Minute, Author: "Bob <bob@assaio.test>"},
		},
		Sessions: []sessionSpec{
			{ID: "s1", Member: "alice", Start: 0, End: hour, Lines: 100},
			{ID: "s2", Member: "bob", Start: 0, End: hour, Lines: 100},
		},
		Expect: map[string]Expectation{
			"s1": {Candidates: []string{"c1", "c2"}, Ambiguous: true},
			"s2": {Candidates: []string{"c1", "c2"}, Ambiguous: true},
		},
	},
	{
		Name: "genuinely-ambiguous",
		Defends: "nothing in the evidence separates these commits, so an engine must keep both " +
			"rather than present a coin flip as a finding",
		Commits: []commitSpec{
			{Tag: "c1", Files: []string{"internal/app/one.go"}, At: hour + 5*time.Minute},
			{Tag: "c2", Files: []string{"internal/app/two.go"}, At: hour + 5*time.Minute},
		},
		Sessions: []sessionSpec{work("s1")},
		Expect:   map[string]Expectation{"s1": {Candidates: []string{"c1", "c2"}, Ambiguous: true}},
	},
	{
		Name:    "manual-correction",
		Defends: "a human's confirmation outranks every heuristic, including a better-scoring one",
		Commits: []commitSpec{
			{Tag: "c1", Files: []string{"internal/app/one.go"}, At: hour + 5*time.Minute},
			{Tag: "c2", Files: []string{"internal/app/two.go"}, At: hour + 5*time.Minute},
		},
		Sessions:    []sessionSpec{work("s1")},
		Corrections: map[string]string{"s1": "c2"},
		Expect:      map[string]Expectation{"s1": {Candidates: []string{"c2"}, Confirmed: "c2"}},
	},
	{
		Name: "replay-after-algorithm-change",
		Defends: "new git data and a new algorithm re-run attribution, and a confirmed link " +
			"survives both rather than being recomputed away",
		Commits: []commitSpec{
			{Tag: "c1", Files: []string{"internal/app/one.go"}, At: hour + 5*time.Minute},
			{Tag: "c2", Files: []string{"internal/app/two.go"}, At: hour + 6*time.Minute},
			// c3 arrives after the correction was made and scores better on proximity: it is
			// the commit a re-run is most likely to prefer, which is the point.
			{Tag: "c3", Files: []string{"internal/app/three.go"}, At: hour + 2*time.Minute},
		},
		Sessions:    []sessionSpec{work("s1")},
		Corrections: map[string]string{"s1": "c1"},
		Expect:      map[string]Expectation{"s1": {Candidates: []string{"c1"}, Confirmed: "c1"}},
	},
}
