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
			if !parser.Answers(tool, declared[i].ID) {
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
