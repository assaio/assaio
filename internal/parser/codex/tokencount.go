package codex

import (
	"encoding/json"
	"fmt"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

type tokenCount struct {
	Type string `json:"type"`
	// Info is a pointer so a rate-limit-only update -- which codex-rs sends as
	// {"type":"token_count","info":null} -- is distinguishable from one reporting zero
	// totals. A nil Info must never reset the running cumulative baseline (st.prev), or
	// the next real token_count would re-count the whole session as one delta.
	Info *struct {
		Total struct {
			Input     int64 `json:"input_tokens"`
			Cached    int64 `json:"cached_input_tokens"`
			Output    int64 `json:"output_tokens"`
			Reasoning int64 `json:"reasoning_output_tokens"`
		} `json:"total_token_usage"`
	} `json:"info"`
}

type totals struct{ input, cached, output, reasoning int64 }

func (st *parseState) applyTokenCount(payload json.RawMessage) {
	rec, ok, err := st.tokenCountRecord(payload)
	if err != nil {
		st.skipped++
		return
	}
	if !ok {
		return
	}
	st.pending.flushInto(&rec)
	st.out = append(st.out, rec)
	st.turn++
}

func (st *parseState) tokenCountRecord(payload json.RawMessage) (usage.Record, bool, error) {
	var tk tokenCount
	if err := json.Unmarshal(payload, &tk); err != nil {
		return usage.Record{}, false, err
	}
	if tk.Type != "token_count" || tk.Info == nil {
		return usage.Record{}, false, nil
	}
	cur := totals{tk.Info.Total.Input, tk.Info.Total.Cached, tk.Info.Total.Output, tk.Info.Total.Reasoning}
	d := totals{cur.input - st.prev.input, cur.cached - st.prev.cached, cur.output - st.prev.output, cur.reasoning - st.prev.reasoning}
	st.prev = cur
	if d.input <= 0 && d.output <= 0 && d.cached <= 0 {
		return usage.Record{}, false, nil
	}
	// input_tokens is the whole prompt and cached_input_tokens the part of it served from
	// cache, so the two stored classes have to add back up to the input delta. Clamping the
	// cached part to it is what holds that: the counters are two independent cumulative
	// numbers, and a turn where cached advanced further than input would otherwise store more
	// prompt tokens than the vendor's own total gained. Unobserved on the audited corpus (0 of
	// 1,686 events) -- this is the invariant, not a correction to a figure.
	cacheRead := parser.Subset(d.cached, parser.NonNeg(d.input))
	return usage.Record{
		Tool:            tool,
		SessionID:       st.session,
		Timestamp:       st.ts,
		Model:           st.model,
		InputTokens:     parser.NonNeg(d.input) - cacheRead,
		CacheReadTokens: cacheRead,
		OutputTokens:    parser.NonNeg(d.output),
		ReasoningTokens: parser.Subset(d.reasoning, parser.NonNeg(d.output)),
		DedupeKey:       fmt.Sprintf("%s:%s:%d", st.fileFP, st.session, st.turn),
		Cwd:             st.cwd,
		Project:         st.project,
		Granularity:     "turn",
	}, true, nil
}
