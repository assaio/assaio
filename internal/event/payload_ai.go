package event

import (
	"errors"
	"fmt"
)

// Usage is what one AI-tool request cost. Counts only: assaio never stores the request.
type Usage struct {
	// Model is the identifier the tool itself reported, "" when its log names none.
	Model            string `json:"model,omitempty"`
	InputTokens      int64  `json:"inputTokens"`
	OutputTokens     int64  `json:"outputTokens"`
	CacheReadTokens  int64  `json:"cacheReadTokens"`
	CacheWriteTokens int64  `json:"cacheWriteTokens"`
	// ReasoningTokens is the thinking portion already included in OutputTokens. It is an
	// informational subset -- adding it to a total double-counts.
	ReasoningTokens int64 `json:"reasoningTokens"`
}

func (Usage) eventType() string { return TypeUsage }

func (u Usage) validate() error {
	if err := nonNegative(map[string]int64{
		"inputTokens": u.InputTokens, "outputTokens": u.OutputTokens,
		"cacheReadTokens": u.CacheReadTokens, "cacheWriteTokens": u.CacheWriteTokens,
		"reasoningTokens": u.ReasoningTokens,
	}); err != nil {
		return err
	}
	if u.ReasoningTokens > u.OutputTokens {
		return fmt.Errorf("reasoningTokens %d exceeds outputTokens %d it is a subset of",
			u.ReasoningTokens, u.OutputTokens)
	}
	return nil
}

// Edit is what one AI-tool request changed. Line and edit counts only -- the paths and the
// diffs they were counted from are read during parsing and never stored.
type Edit struct {
	LinesAdded   int64 `json:"linesAdded"`
	LinesRemoved int64 `json:"linesRemoved"`
	Edits        int64 `json:"edits"`
	// ReworkLines is added lines a later edit in the same transcript removed again -- a
	// churn proxy, derived during parsing.
	ReworkLines int64 `json:"reworkLines"`
}

func (Edit) eventType() string { return TypeEdit }

// hasActivity reports whether this observation says anything at all. A source with no edit
// extraction produces no edit observation rather than one full of zeros, so "no activity
// recorded" and "no activity happened" stay distinguishable.
func (e Edit) hasActivity() bool {
	return e.LinesAdded > 0 || e.LinesRemoved > 0 || e.Edits > 0 || e.ReworkLines > 0
}

func (e Edit) validate() error {
	if err := nonNegative(map[string]int64{
		"linesAdded": e.LinesAdded, "linesRemoved": e.LinesRemoved,
		"edits": e.Edits, "reworkLines": e.ReworkLines,
	}); err != nil {
		return err
	}
	if !e.hasActivity() {
		return errors.New("edit observation carries no activity")
	}
	return nil
}
