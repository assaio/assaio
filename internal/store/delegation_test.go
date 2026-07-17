package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func TestDelegationSplitsSubAgentFromTotalTokens(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "agent:ag1",
			InputTokens: 100, OutputTokens: 50, CacheReadTokens: 10, CacheWriteTokens: 5, ReasoningTokens: 1,
		},
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "turn-1",
			InputTokens: 200, OutputTokens: 100,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	sub, total, err := s.Delegation(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if wantSub := int64(100 + 50 + 10 + 5 + 1); sub != wantSub {
		t.Fatalf("sub = %d, want %d", sub, wantSub)
	}
	if wantTotal := int64(166 + 300); total != wantTotal {
		t.Fatalf("total = %d, want %d", total, wantTotal)
	}
}

func TestDelegationExcludesRowsBeforeSince(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	old := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: old, Model: "m", DedupeKey: "agent:ag1", InputTokens: 100},
	})
	if err != nil {
		t.Fatal(err)
	}

	sub, total, err := s.Delegation(ctx, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if sub != 0 || total != 0 {
		t.Fatalf("sub=%d total=%d, want 0/0 for a row entirely before since", sub, total)
	}
}

// TestDelegationDedupeKeyPrefixMatchIsExact asserts the LIKE 'agent:%' match only catches
// the real "agent:" prefix, not a dedupe_key that merely contains "agent" mid-string.
func TestDelegationDedupeKeyPrefixMatchIsExact(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "not-agent:ag1", InputTokens: 42},
	})
	if err != nil {
		t.Fatal(err)
	}

	sub, total, err := s.Delegation(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if sub != 0 {
		t.Fatalf("sub = %d, want 0: a dedupe_key not starting with \"agent:\" must not count", sub)
	}
	if total != 42 {
		t.Fatalf("total = %d, want 42", total)
	}
}

// TestDelegationMatchesMemberPrefixedAgentKey guards the central-store case: handleUsage
// (internal/server/handlers.go) rewrites a synced record's dedupe_key to "<member>:" +
// original, so a sub-agent's "agent:ag1" key arrives here as "alice:agent:ag1".
// Delegation must still recognize it as sub-agent delegation, not silently undercount
// every synced team member's share to zero.
func TestDelegationMatchesMemberPrefixedAgentKey(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "alice:agent:ag1",
			InputTokens: 100, OutputTokens: 50, CacheReadTokens: 10, CacheWriteTokens: 5, ReasoningTokens: 1,
			Member: "alice",
		},
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "alice:turn-1",
			InputTokens: 200, OutputTokens: 100, Member: "alice",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	sub, total, err := s.Delegation(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if wantSub := int64(100 + 50 + 10 + 5 + 1); sub != wantSub {
		t.Fatalf("sub = %d, want %d (member-prefixed agent key must still count as delegation)", sub, wantSub)
	}
	if wantTotal := int64(166 + 300); total != wantTotal {
		t.Fatalf("total = %d, want %d", total, wantTotal)
	}
}

// TestDelegationExcludesNonClaudeAgentPrefixedKey guards against a coincidental
// dedupe_key collision: a non-claude-code tool's own key scheme (e.g. codex's
// "<session>:<turn>") can produce "agent:0" if a session or task happens to be literally
// named "agent" -- that must never be counted as Claude Task delegation.
func TestDelegationExcludesNonClaudeAgentPrefixedKey(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "codex", SessionID: "agent", Timestamp: ts, Model: "m", DedupeKey: "agent:0", InputTokens: 42},
	})
	if err != nil {
		t.Fatal(err)
	}

	sub, total, err := s.Delegation(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if sub != 0 {
		t.Fatalf("sub = %d, want 0: a non-claude-code row must never count as Claude sub-agent delegation", sub)
	}
	if total != 42 {
		t.Fatalf("total = %d, want 42", total)
	}
}

func TestDelegationEmptyStoreIsZeroNotError(t *testing.T) {
	s := newStore(t)
	sub, total, err := s.Delegation(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if sub != 0 || total != 0 {
		t.Fatalf("sub=%d total=%d, want 0/0 on an empty store", sub, total)
	}
}
