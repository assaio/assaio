// Package pricing loads LiteLLM-format price tables and prices usage.Record values.
package pricing

import (
	"embed"
	"encoding/json"
	"io"
	"sync"

	"github.com/assaio/assaio/internal/usage"
)

// litellm.json is a snapshot of LiteLLM's model_prices_and_context_window.json (MIT, see NOTICE); refresh by re-downloading from the LiteLLM repo.
//
//go:embed litellm.json
var vendored embed.FS

// Price holds per-token costs in USD for one model.
type Price struct{ Input, Output, CacheWrite, CacheRead float64 }

// Table maps a model name to its Price.
type Table map[string]Price

// litellmEntry mirrors the fields we use from LiteLLM's price file. Cache-write is a
// pointer because OpenAI entries set it to null.
type litellmEntry struct {
	Input      float64  `json:"input_cost_per_token"`
	Output     float64  `json:"output_cost_per_token"`
	CacheWrite *float64 `json:"cache_creation_input_token_cost"`
	CacheRead  *float64 `json:"cache_read_input_token_cost"`
}

// loadOnce guards cachedTable/cachedErr: Load parses the embedded price table at most
// once per process (it's a 1.5MB file re-parsed on every call otherwise -- a cost paid on
// every dashboard build and every server request), and every call returns the cached result.
var (
	loadOnce    sync.Once
	cachedTable Table
	cachedErr   error
)

// Load reads the vendored LiteLLM price table embedded in the binary, parsing it once per
// process; every call after the first returns the same cached Table.
func Load() (Table, error) {
	loadOnce.Do(func() {
		f, err := vendored.Open("litellm.json")
		if err != nil {
			cachedErr = err
			return
		}
		defer func() { _ = f.Close() }()
		cachedTable, cachedErr = LoadReader(f)
	})
	return cachedTable, cachedErr
}

// LoadReader parses a LiteLLM-format price table from r.
func LoadReader(r io.Reader) (Table, error) {
	var raw map[string]litellmEntry
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, err
	}
	t := make(Table, len(raw))
	for name, e := range raw {
		p := Price{Input: e.Input, Output: e.Output}
		if e.CacheWrite != nil {
			p.CacheWrite = *e.CacheWrite
		}
		if e.CacheRead != nil {
			p.CacheRead = *e.CacheRead
		}
		t[name] = p
	}
	return t, nil
}

// Cost computes the USD cost of r using t, trying the exact model name then its
// normalized form. ok is false when the model is not priced.
func (t Table) Cost(r *usage.Record) (float64, bool) {
	return t.CostTokens(r.Model, r.InputTokens, r.OutputTokens, r.CacheWriteTokens, r.CacheReadTokens)
}

// CostTokens computes the USD cost of the given token counts for model using t, trying
// the exact model name then its normalized form. ok is false when the model is not priced.
func (t Table) CostTokens(model string, in, out, cacheWrite, cacheRead int64) (float64, bool) {
	p, ok := t[model]
	if !ok {
		p, ok = t[NormalizeModel(model)]
	}
	if !ok {
		return 0, false
	}
	cost := float64(in)*p.Input +
		float64(out)*p.Output +
		float64(cacheWrite)*p.CacheWrite +
		float64(cacheRead)*p.CacheRead
	return cost, true
}
