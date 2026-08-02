package signal

import (
	"sort"

	"github.com/assaio/assaio/internal/parser"
)

// The support verdicts, from a signal every token can answer to one nothing in this window
// can. Partial is the interesting one: it is where a figure is real but describes less of the
// window than a reader would assume.
const (
	Full    = "full"
	Partial = "partial"
	None    = "none"
)

// Support is what one signal can honestly say about the data actually in the store.
type Support struct {
	Signal Signal
	// Share is the fraction of the window's tokens from sources that can answer it, 0..1.
	Share float64
	// Verdict is Full, Partial or None.
	Verdict string
	// Sources names the tools that can answer this signal, alphabetically.
	Sources []string
}

// Coverage answers, per signal, whether the data in hand can support it -- computed from the
// window's real token mix and what each source declares it answers, never from a claim the
// catalog makes about itself. byTool maps a tool name to its tokens in the window
// (analyze.TokensByTool).
func Coverage(byTool map[string]int64) []Support {
	var total int64
	tools := make([]string, 0, len(byTool))
	for tool, n := range byTool {
		total += n
		tools = append(tools, tool)
	}
	sort.Strings(tools)

	declared := Catalog()
	out := make([]Support, 0, len(declared))
	for i := range declared {
		var capable int64
		var sources []string
		for _, tool := range tools {
			if !answers(tool, declared[i].ID) {
				continue
			}
			capable += byTool[tool]
			sources = append(sources, tool)
		}
		out = append(out, Support{
			Signal:  declared[i],
			Share:   share(capable, total),
			Verdict: verdict(capable, total),
			Sources: sources,
		})
	}
	return out
}

// answers reports whether tool's parser produces the data behind id. A tool outside the depth
// matrix is an out-of-tree exec plugin, and that protocol carries tokens and nothing else --
// so it answers the token signals and no others, which is what it can support rather than a
// guess about its author.
func answers(tool, id string) bool {
	d, known := parser.DepthOf(tool)
	if !known {
		return pluginAnswerable[id]
	}
	for _, answered := range d.Answers {
		if answered == id {
			return true
		}
	}
	return false
}

// pluginAnswerable is what the exec parser protocol's record shape can carry: tokens and the
// session identity around them. It has no activity or attribution fields at all (ADR 0003).
var pluginAnswerable = map[string]bool{
	"ai.tokens.total": true, "ai.tokens.input": true, "ai.tokens.output": true,
	"ai.tokens.cache_read": true, "ai.tokens.cache_write": true, "ai.tokens.reasoning": true,
	"ai.cost.estimated": true, "ai.sessions.count": true,
	"ai.turns.count": true, "ai.session.active_minutes": true,
}

func share(capable, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(capable) / float64(total)
}

func verdict(capable, total int64) string {
	switch {
	case total <= 0 || capable == 0:
		return None
	case capable == total:
		return Full
	default:
		return Partial
	}
}
