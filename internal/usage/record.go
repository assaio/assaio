// Package usage defines the normalized representation of AI-tool usage events.
package usage

import "time"

// Record is one normalized usage event from any AI tool. Session-level provenance:
// it derives from local logs/hooks, never from daily vendor aggregates.
type Record struct {
	Tool             string
	SessionID        string
	Timestamp        time.Time
	Model            string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	// ReasoningTokens is the thinking/reasoning portion that is already included in
	// OutputTokens (an informational subset, billed at the output rate). It is never
	// added to a token total and never priced on its own -- doing either double-counts.
	ReasoningTokens int64
	DedupeKey       string
	// Member is a pseudonymized author/agent id, set by the server from a sync push;
	// "" for purely-local usage. Never set by a parser.
	Member string
	// Cwd is the session's full working-directory path, exactly as the tool's log
	// reports it. TRANSIENT: ingest reads it only to resolve Project/Subpath
	// (internal/projectid) and never persists it — PRIVACY.md promises the store holds
	// only a basename, never a full path. "" if the log carries no cwd.
	Cwd string `json:"-"`
	// Project is the basename of the git repository root containing Cwd, resolved at
	// ingest time. Parsers set it only as a fallback (their own leaf-directory guess,
	// e.g. filepath.Base(cwd)); ingest overwrites it whenever Cwd resolves to a
	// repository root, so a monorepo's subdirectories roll up to one project.
	Project string
	// Subpath is Cwd's path relative to the resolved repository root (e.g.
	// "apps/mobile"), or "" at the root or when unresolved. Set by ingest, never by a
	// parser; always relative, never an absolute path.
	Subpath string
	// GitBranch is the branch name if the log carries it, else "".
	GitBranch string
	// Entrypoint is how the tool was invoked (e.g. "cli", "sdk-py"), else "".
	Entrypoint string
	// Granularity is "turn" for per-request records or "session" for session-aggregate
	// sources; session-level data must never silently masquerade as per-turn.
	Granularity string
	// LinesAdded is AI-added code lines in this record's edits (diff "+" markers); zero
	// when unknown. Populated by the Claude Code and Codex parsers; 0 for tools without
	// edit-log extraction. assaio counts lines, never stores the code itself.
	LinesAdded int64
	// LinesRemoved is AI-removed code lines in this record's edits (diff "-" markers);
	// zero when unknown. Populated by the Claude Code and Codex parsers; 0 for tools
	// without edit-log extraction.
	LinesRemoved int64
	// Edits is the count of edit-type tool calls (Edit/Write/NotebookEdit/MultiEdit) in
	// this turn; zero when unknown. Populated by the Claude Code and Codex parsers; 0
	// for tools without edit-log extraction.
	Edits int64
	// ToolCalls is the count of all tool_use blocks in this turn; zero when unknown.
	// Populated by the Claude Code and Codex parsers; 0 for tools without edit-log
	// extraction.
	ToolCalls int64
	// Rejected is the count of tool-use denials attributed to this turn; zero when
	// unknown. Populated by the Claude Code parser only -- Codex's rollout logs don't
	// surface tool-use denials the way Claude Code's do.
	Rejected int64
	// Compactions is the count of context-compaction events attributed to this turn: a
	// context-strain signal. Populated by the Claude Code and Codex parsers; 0 for tools
	// without edit-log extraction.
	Compactions int64
	// ReworkLines is AI-added code lines later removed by a subsequent edit to the same
	// file within this transcript -- a rework/thrash proxy for "AI wrote code that
	// didn't stick." Populated by the Claude Code and Codex parsers (both share the
	// internal/parser.Rework helper); 0 for tools without edit-log extraction. The file
	// path used to detect this is read transiently during parsing and never stored.
	ReworkLines int64
}
