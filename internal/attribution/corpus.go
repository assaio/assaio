// Package attribution holds the conformance corpus that defines what an honest
// session-to-commit link is, before any engine or UI exists to make one. Each scenario
// builds a real git repository and the sessions that ran against it, and states what any
// engine must say about them -- most importantly, where it must refuse to choose.
//
// The corpus deliberately speaks the smallest vocabulary that honesty can be judged in:
// which commits a session may be linked to, and whether the link stayed ambiguous. An
// engine's own edge carries its method, confidence, evidence and the alternatives it beat
// (B85); reducing its answer to this shape is the engine's job, so the corpus cannot bake
// one engine's types into the definition of correct.
package attribution

// Expectation is what any engine must say about one session.
type Expectation struct {
	// Candidates names the commits, by scenario tag, a link may point at. Empty means the
	// session must stay unattributed: no evidence in the fixture reaches any commit.
	Candidates []string
	// Ambiguous requires the engine to keep every candidate rather than pick one. This is
	// the assertion the corpus exists for: nothing in the fixture distinguishes them, so a
	// single confident answer is a fabricated one.
	Ambiguous bool
	// Confirmed is a commit tag a human confirmed by hand. It must win over whatever the
	// evidence suggests, and must still win after the algorithm changes.
	Confirmed string
}

// Link is one session's answer, reduced to what the corpus judges: the commits an engine
// linked it to, and whether it reported the link as ambiguous.
type Link struct {
	Commits   []string
	Ambiguous bool
}

// Links is an engine's whole answer for one scenario, keyed by session id. A session absent
// from the map is unattributed, which for some scenarios is the correct answer.
type Links map[string]Link

// Scenario is one attribution situation: why it exists, the data it builds, and what any
// engine must conclude from it.
type Scenario struct {
	Name string
	// Defends is the honesty property this scenario protects, in one line. It is printed
	// with any violation, so a failure says what was broken rather than only what differed.
	Defends string
	Commits []commitSpec
	// Sessions are the AI sessions that ran against the repository, in the same window.
	Sessions []sessionSpec
	// Corrections are the links a human confirmed by hand, session id to commit tag.
	Corrections map[string]string
	Expect      map[string]Expectation
}

// Corpus is every scenario, in the order they are worth reading: the shapes a link can
// take, then the noise an engine has to resist, then the cases where it must not answer.
func Corpus() []Scenario {
	out := make([]Scenario, len(scenarios))
	copy(out, scenarios)
	return out
}

// Get returns the scenario registered under name.
func Get(name string) (Scenario, bool) {
	for i := range scenarios {
		if scenarios[i].Name == name {
			return scenarios[i], true
		}
	}
	return Scenario{}, false
}
