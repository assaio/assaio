package label

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Sources a rule may read. Each is something the store already stamped on the session
// because a tool wrote it down -- never anything inferred from what was said.
const (
	SourceBranch     = "branch"
	SourceSkill      = "skill"
	SourceAgent      = "agent"
	SourceEntrypoint = "entrypoint"
)

// Sources lists them in the order a suggestion reports ties, so two installs reading the
// same session derive the same answer.
var Sources = []string{SourceBranch, SourceSkill, SourceAgent, SourceEntrypoint}

// Rule derives one axis value from one signal source. Match is an RE2 pattern -- Go's
// engine has no backtracking, so a pathological pattern costs time linear in the input
// rather than opening a denial of service on the person's own machine.
type Rule struct {
	Source string `yaml:"source"`
	Match  string `yaml:"match"`
	Axis   string `yaml:"axis"`
	Value  string `yaml:"value"`
}

// Signals is what one session's rows recorded, keyed by source. A source the session
// carries nothing for is absent, which no rule can match -- silence never derives a label.
type Signals map[string][]string

// Suggestion is one axis's derived value and the evidence for it, so a person reading
// "feature" can see it came from a branch called feat/rules-engine rather than a guess.
type Suggestion struct {
	Axis   string
	Value  string
	Source string
	Reason string
}

// Engine applies a compiled rule set. Compiling once is what keeps a suggestion over
// thousands of sessions from recompiling the same pattern per row.
type Engine struct {
	rules []compiled
}

type compiled struct {
	Rule
	re *regexp.Regexp
}

// NewEngine compiles rules, rejecting any that names an unknown source, an unknown axis, a
// value outside that axis's closed vocabulary, or a pattern that does not compile. A rule
// set is configuration a person wrote, so it fails loudly at load rather than deriving
// nothing at midnight.
func NewEngine(rules []Rule) (*Engine, error) {
	e := &Engine{rules: make([]compiled, 0, len(rules))}
	for i, r := range rules {
		if !validSource(r.Source) {
			return nil, fmt.Errorf("rule %d: unknown source %q (want %s)", i+1, r.Source, strings.Join(Sources, "|"))
		}
		if _, ok := Axes[r.Axis]; !ok {
			return nil, fmt.Errorf("rule %d: unknown axis %q (want %s)", i+1, r.Axis, strings.Join(Names, "|"))
		}
		if r.Value == "" || !Valid(r.Axis, r.Value) {
			return nil, fmt.Errorf("rule %d: %q is not a %s (want %s)", i+1, r.Value, r.Axis, Values(r.Axis))
		}
		re, err := regexp.Compile(r.Match)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i+1, err)
		}
		e.rules = append(e.rules, compiled{Rule: r, re: re})
	}
	return e, nil
}

// Derive returns at most one suggestion per axis. Rules that agree collapse to one answer;
// rules that disagree yield nothing for that axis, because a session whose branch says
// bugfix and whose skill says test is genuinely ambiguous and a coin flip would be worse
// than the blank the person already has.
func (e *Engine) Derive(sig Signals) []Suggestion {
	hits := map[string][]Suggestion{}
	for _, c := range e.rules {
		for _, v := range sig[c.Source] {
			if v == "" || !c.re.MatchString(v) {
				continue
			}
			hits[c.Axis] = append(hits[c.Axis], Suggestion{
				Axis: c.Axis, Value: c.Value, Source: c.Source,
				Reason: fmt.Sprintf("%s %q matches %s", c.Source, v, c.Match),
			})
			break
		}
	}
	out := make([]Suggestion, 0, len(hits))
	for _, axis := range Names {
		if s, ok := agreed(hits[axis]); ok {
			out = append(out, s)
		}
	}
	return out
}

// agreed collapses one axis's hits: identical values become a single suggestion naming
// every source that agreed, and a disagreement becomes no suggestion at all.
func agreed(hits []Suggestion) (Suggestion, bool) {
	if len(hits) == 0 {
		return Suggestion{}, false
	}
	sources := make([]string, 0, len(hits))
	for _, h := range hits {
		if h.Value != hits[0].Value {
			return Suggestion{}, false
		}
		sources = append(sources, h.Source)
	}
	if len(hits) == 1 {
		return hits[0], true
	}
	sort.Strings(sources)
	return Suggestion{
		Axis: hits[0].Axis, Value: hits[0].Value, Source: strings.Join(sources, "+"),
		Reason: hits[0].Reason + fmt.Sprintf(" (and %d more agreeing)", len(hits)-1),
	}, true
}

func validSource(s string) bool {
	for _, known := range Sources {
		if known == s {
			return true
		}
	}
	return false
}
