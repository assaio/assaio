package server

import (
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// SyncRecordV2 is the closed sync wire. Member and GitBranch have no field, so a
// new usage.Record field cannot begin leaving the machine without a wire change.
type SyncRecordV2 struct {
	Tool               string
	SessionID          string
	Timestamp          time.Time
	Model              string
	InputTokens        int64
	OutputTokens       int64
	CacheReadTokens    int64
	CacheWriteTokens   int64
	CacheWrite1hTokens int64
	CacheMissReason    string
	ReasoningTokens    int64
	DedupeKey          string
	Project            string
	Subpath            string
	Entrypoint         string
	Granularity        string
	LinesAdded         int64
	LinesRemoved       int64
	Edits              int64
	ToolCalls          int64
	Rejected           int64
	Compactions        int64
	ToolReads          int64
	ToolSearches       int64
	ToolCommands       int64
	ToolWrites         int64
	ToolOther          int64
	ToolErrors         int64
	Sidechain          int64
	Skill              string
	Agent              string
	ReworkLines        int64
}

// NewSyncRecordV2 projects one local record onto the field-level sync allowlist.
func NewSyncRecordV2(r *usage.Record) SyncRecordV2 {
	return SyncRecordV2{
		Tool: r.Tool, SessionID: r.SessionID, Timestamp: r.Timestamp, Model: r.Model,
		InputTokens: r.InputTokens, OutputTokens: r.OutputTokens,
		CacheReadTokens: r.CacheReadTokens, CacheWriteTokens: r.CacheWriteTokens,
		CacheWrite1hTokens: r.CacheWrite1hTokens, CacheMissReason: r.CacheMissReason,
		ReasoningTokens: r.ReasoningTokens, DedupeKey: r.DedupeKey,
		Project: r.Project, Subpath: r.Subpath, Entrypoint: r.Entrypoint,
		Granularity: r.Granularity, LinesAdded: r.LinesAdded, LinesRemoved: r.LinesRemoved,
		Edits: r.Edits, ToolCalls: r.ToolCalls, Rejected: r.Rejected,
		Compactions: r.Compactions, ToolReads: r.ToolReads, ToolSearches: r.ToolSearches,
		ToolCommands: r.ToolCommands, ToolWrites: r.ToolWrites, ToolOther: r.ToolOther,
		ToolErrors: r.ToolErrors, Sidechain: r.Sidechain, Skill: r.Skill,
		Agent: r.Agent, ReworkLines: r.ReworkLines,
	}
}

func (r *SyncRecordV2) usageRecord() usage.Record {
	return usage.Record{
		Tool: r.Tool, SessionID: r.SessionID, Timestamp: r.Timestamp, Model: r.Model,
		InputTokens: r.InputTokens, OutputTokens: r.OutputTokens,
		CacheReadTokens: r.CacheReadTokens, CacheWriteTokens: r.CacheWriteTokens,
		CacheWrite1hTokens: r.CacheWrite1hTokens, CacheMissReason: r.CacheMissReason,
		ReasoningTokens: r.ReasoningTokens, DedupeKey: r.DedupeKey,
		Project: r.Project, Subpath: r.Subpath, Entrypoint: r.Entrypoint,
		Granularity: r.Granularity, LinesAdded: r.LinesAdded, LinesRemoved: r.LinesRemoved,
		Edits: r.Edits, ToolCalls: r.ToolCalls, Rejected: r.Rejected,
		Compactions: r.Compactions, ToolReads: r.ToolReads, ToolSearches: r.ToolSearches,
		ToolCommands: r.ToolCommands, ToolWrites: r.ToolWrites, ToolOther: r.ToolOther,
		ToolErrors: r.ToolErrors, Sidechain: r.Sidechain, Skill: r.Skill,
		Agent: r.Agent, ReworkLines: r.ReworkLines,
	}
}
