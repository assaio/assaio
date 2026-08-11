package label

// DefaultRules derives a task class from branch naming conventions only.
//
// Deliberately nothing about skills, sub-agents or entrypoints ships here, though the engine
// reads all four: a skill name is one person's toolchain, and a default rule over it would
// encode whoever wrote it. Branch prefixes are the one convention that is widespread, tool
// independent and chosen by the author before any assistant ran.
//
// Every pattern demands a separator, so `fixture/…` is not a bugfix and `features` is not a
// feature. A branch with no convention -- `main`, `develop`, `JIRA-4821` -- matches nothing
// and yields no label, which is the intended answer rather than a missing one.
var DefaultRules = []Rule{
	{Source: SourceBranch, Match: `(?i)^(feat|feature)s?[/_-]`, Axis: Task, Value: "feature"},
	{Source: SourceBranch, Match: `(?i)^(fix|bugfix|hotfix|bug)[/_-]`, Axis: Task, Value: "bugfix"},
	{Source: SourceBranch, Match: `(?i)^(test|tests|testing)[/_-]`, Axis: Task, Value: "test"},
	{Source: SourceBranch, Match: `(?i)^(refactor|refac|cleanup)[/_-]`, Axis: Task, Value: "refactor"},
	{Source: SourceBranch, Match: `(?i)^(doc|docs)[/_-]`, Axis: Task, Value: "docs"},
	{Source: SourceBranch, Match: `(?i)^(spike|research|poc|experiment|explore)[/_-]`, Axis: Task, Value: "research"},
	{Source: SourceBranch, Match: `(?i)^(review|audit)[/_-]`, Axis: Task, Value: "review"},
}
