package docs

import "github.com/assaio/assaio/internal/parser"

// Source is one row of the depth matrix: what a tool can actually tell you, per signal rather
// than per checkmark. Out-of-tree exec parsers are absent for the reason parser.DepthOf omits
// them -- their capability is whatever their author implemented, so it is read from the records
// they emit rather than promised by a table assaio maintains.
type Source struct {
	Tool string `json:"tool"`
	Tier string `json:"tier"`
	// Tokens, Activity and Attribution are the tier table's one-bit summaries. Answers is the
	// per-signal truth and the two are not interchangeable: "has activity" reads true for a
	// source that reports changed lines and no edit count, no tool calls and no turns.
	Tokens      bool     `json:"tokens"`
	Activity    bool     `json:"activity"`
	Attribution bool     `json:"attribution"`
	Answers     []string `json:"answers"`
	Gaps        []string `json:"gaps"`
}

func sources() []Source {
	tools := parser.Tools()
	out := make([]Source, 0, len(tools))
	for _, tool := range tools {
		d, ok := parser.DepthOf(tool)
		if !ok {
			continue
		}
		out = append(out, Source{
			Tool:        d.Tool,
			Tier:        d.Tier,
			Tokens:      d.Tokens,
			Activity:    d.Activity,
			Attribution: d.Attribution,
			Answers:     parser.SignalsAnsweredBy(tool),
			Gaps:        list(d.Gaps),
		})
	}
	return out
}

func sourcesAnswering(id string) []string { return list(parser.SourcesAnswering(id)) }

// list renders an absent set as [] rather than null. A consumer that ranges over the result
// then reads "nothing answers this" instead of failing to decode -- the same choice the metric
// wire makes for its empty aggregates.
func list(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
