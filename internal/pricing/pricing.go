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

// Price holds per-token costs in USD for one model. CacheWrite1h is the rate for a write
// that bought a 1-hour cache lifetime; it falls back to CacheWrite for a model whose table
// entry publishes only one cache-write rate, which is the vendor charging one rate rather
// than a tier being free.
type Price struct{ Input, Output, CacheWrite, CacheRead, CacheWrite1h float64 }

// Table maps a model name to its Price.
type Table map[string]Price

// Tokens is the billable token counts of one model's usage. CacheWrite1h is the portion of
// CacheWrite that bought a 1-hour cache lifetime: a subset priced at its own rate, never
// added to a total. Passing the counts as a value keeps a new billing tier from growing
// every call site's argument list.
type Tokens struct {
	In, Out, CacheWrite, CacheRead, CacheWrite1h int64
}

// litellmEntry mirrors the fields we use from LiteLLM's price file. Cache-write is a
// pointer because OpenAI entries set it to null.
type litellmEntry struct {
	Input        float64  `json:"input_cost_per_token"`
	Output       float64  `json:"output_cost_per_token"`
	CacheWrite   *float64 `json:"cache_creation_input_token_cost"`
	CacheWrite1h *float64 `json:"cache_creation_input_token_cost_above_1hr"`
	CacheRead    *float64 `json:"cache_read_input_token_cost"`
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
		p.CacheWrite1h = p.CacheWrite
		if e.CacheWrite1h != nil {
			p.CacheWrite1h = *e.CacheWrite1h
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
	return t.CostTokens(r.Model, Tokens{
		In: r.InputTokens, Out: r.OutputTokens,
		CacheWrite: r.CacheWriteTokens, CacheRead: r.CacheReadTokens,
		CacheWrite1h: r.CacheWrite1hTokens,
	})
}

// CostTokens computes the USD cost of tk for model using t, trying the exact model name
// then its normalized form. ok is false when the model is not priced. A cache write is
// split at its 1-hour portion and each part billed at its own rate; a source that does not
// report the tier leaves that portion zero and is priced exactly as before.
func (t Table) CostTokens(model string, tk Tokens) (float64, bool) {
	p, ok := t[model]
	if !ok {
		p, ok = t[NormalizeModel(model)]
	}
	if !ok {
		return 0, false
	}
	// The 1-hour portion is clamped to the write it is part of: a stored subset larger than
	// its whole would otherwise bill a negative 5-minute remainder.
	long := min(max(tk.CacheWrite1h, 0), max(tk.CacheWrite, 0))
	cost := float64(tk.In)*p.Input +
		float64(tk.Out)*p.Output +
		float64(tk.CacheWrite-long)*p.CacheWrite +
		float64(long)*p.CacheWrite1h +
		float64(tk.CacheRead)*p.CacheRead
	return cost, true
}
