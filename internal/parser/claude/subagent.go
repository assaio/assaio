package claude

import (
	"path/filepath"
	"strings"

	"github.com/assaio/assaio/internal/usage"
)

// agentDedupePrefix namespaces a completed sub-agent aggregate's DedupeKey, "agent:<id>".
const agentDedupePrefix = "agent:"

// SubagentID extracts <id> from a discovered subagents/agent-<id>.jsonl path.
func SubagentID(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	return strings.TrimPrefix(base, "agent-")
}

// CoveredAgents is the set of sub-agent ids that have their own transcript file, so
// SuppressCovered can drop the parent's redundant aggregate for each.
func CoveredAgents(subagentFiles []string) map[string]struct{} {
	covered := make(map[string]struct{}, len(subagentFiles))
	for _, f := range subagentFiles {
		if id := SubagentID(f); id != "" {
			covered[id] = struct{}{}
		}
	}
	return covered
}

// SuppressCovered drops any completed sub-agent aggregate ("agent:<id>") whose <id> has
// its own transcript file: that file holds the full per-turn usage the parent aggregate
// only summarizes as its last turn, so counting both would both double-count and, worse,
// undercount. An aggregate for an id with no file is kept, so pre-subagents-file
// transcripts still contribute their sub-agent usage.
func SuppressCovered(recs []usage.Record, covered map[string]struct{}) []usage.Record {
	if len(covered) == 0 {
		return recs
	}
	out := recs[:0]
	for i := range recs {
		if id, ok := strings.CutPrefix(recs[i].DedupeKey, agentDedupePrefix); ok {
			if _, dup := covered[id]; dup {
				continue
			}
		}
		out = append(out, recs[i])
	}
	return out
}
