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
	// CacheWrite1hTokens is the portion of CacheWriteTokens that bought a 1-hour cache
	// lifetime rather than the default 5-minute one; a subset, never added to a total, and
	// priced at its own higher rate. 0 for sources that do not report the tier -- which is
	// indistinguishable from "all writes were 5-minute", so any figure over it declares its
	// own coverage.
	CacheWrite1hTokens int64
	// CacheMissReason is why the vendor could not serve this turn's prompt from cache, from
	// the tool's own closed vocabulary (Claude Code: messages_changed, model_changed,
	// previous_message_not_found, system_changed, tools_changed, unavailable); "" when the
	// turn hit cache or the source reports no reason. A category label, never content.
	CacheMissReason string
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
	// ToolReads, ToolSearches, ToolCommands, ToolWrites, and ToolOther split this turn's
	// ToolCalls by what each call was for -- reading a file, searching, running a command,
	// writing code, or anything else. They sum to ToolCalls for tools whose logs name their
	// tool calls (Claude Code, Codex); all zero for tools that don't, which is why any
	// metric over them reports its own coverage. Only the category is derived: the tool
	// name is classified during parsing and never stored.
	ToolReads    int64
	ToolSearches int64
	ToolCommands int64
	ToolWrites   int64
	ToolOther    int64
	// ToolErrors is the count of this turn's tool calls that returned an error -- a
	// friction signal distinct from Rejected, which is a human declining a call. Populated
	// by the Claude Code and Codex parsers; 0 for tools whose logs don't mark failures.
	ToolErrors int64
	// Sidechain is 1 when this turn ran inside a sub-agent, read from the log's own marker
	// rather than inferred from DedupeKey; 0 for a main-loop turn or a tool that has no
	// sub-agents. Claude Code only today.
	Sidechain int64
	// Skill is the skill the tool attributed this turn to (e.g. "code-review"), "" when
	// none or unsupported. A category label the tool itself assigned -- never a prompt or
	// any content. Claude Code only today.
	Skill string
	// Agent is the sub-agent type this turn ran as (e.g. "general-purpose"), "" for
	// main-loop turns or tools without sub-agents. Category label only. Claude Code only.
	Agent string
	// ReworkLines is AI-added code lines later removed by a subsequent edit to the same
	// file within this transcript -- a rework/thrash proxy for "AI wrote code that
	// didn't stick." Populated by the Claude Code and Codex parsers (both share the
	// internal/parser.Rework helper); 0 for tools without edit-log extraction. The file
	// path used to detect this is read transiently during parsing and never stored.
	ReworkLines int64
}
