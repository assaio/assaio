// Package cline parses Cline task directories into normalized usage records.
//
// Cline stores its own computed cost per request (ClineApiReqInfo.cost). assaio does
// not carry that value: Record has no Cost field by design, so cost is always computed
// from tokens at report time for cross-tool consistency. Cline's stored cost is useful
// only as an external check on assaio's pricing tables, not as an input.
package cline

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

const tool = "cline"

type uiMessage struct {
	Type string `json:"type"`
	Say  string `json:"say"`
	Text string `json:"text"`
	TS   int64  `json:"ts"`
}

type apiReqInfo struct {
	TokensIn    int64    `json:"tokensIn"`
	TokensOut   int64    `json:"tokensOut"`
	CacheWrites int64    `json:"cacheWrites"`
	CacheReads  int64    `json:"cacheReads"`
	Cost        *float64 `json:"cost"`
}

type taskMetadata struct {
	ModelUsage []modelUsageEntry `json:"model_usage"`
}

type modelUsageEntry struct {
	TS      int64  `json:"ts"`
	ModelID string `json:"model_id"`
}

// ParseTask reads one task's ui_messages.json and its task_metadata.json (model may be
// "" if unavailable) into usage records. Only say == "api_req_started" entries carry
// usage; their text field is a JSON-encoded ClineApiReqInfo, not a plain string. Model
// is resolved per request from model_usage by carrying forward the entry with the
// latest ts not after the request's ts, since Cline can switch models mid-task. skipped
// counts api_req_started entries whose text field failed to unmarshal. DedupeKey is
// taskID:index with no per-file fingerprint (unlike gemini/codex): taskID is the task
// directory's basename, which Cline assigns once and never reuses for another directory,
// so -- unlike a resumable session id -- it cannot collide across two different files.
func ParseTask(uiMessages io.Reader, taskID string, meta taskMetadata) ([]usage.Record, int, error) {
	var msgs []uiMessage
	if err := json.NewDecoder(uiMessages).Decode(&msgs); err != nil {
		return nil, 0, err
	}
	models := sortedModelUsage(meta.ModelUsage)

	var out []usage.Record
	var skipped int
	index := 0
	for _, m := range msgs {
		if m.Type != "say" || m.Say != "api_req_started" || m.Text == "" {
			continue
		}
		var info apiReqInfo
		if err := json.Unmarshal([]byte(m.Text), &info); err != nil {
			skipped++
			continue
		}
		if info.TokensIn == 0 && info.TokensOut == 0 {
			continue
		}
		out = append(out, usage.Record{
			Tool:             tool,
			SessionID:        taskID,
			Timestamp:        millisToTime(m.TS),
			Model:            modelAt(models, m.TS),
			InputTokens:      parser.NonNeg(info.TokensIn),
			OutputTokens:     parser.NonNeg(info.TokensOut),
			CacheWriteTokens: parser.NonNeg(info.CacheWrites),
			CacheReadTokens:  parser.NonNeg(info.CacheReads),
			DedupeKey:        fmt.Sprintf("%s:%d", taskID, index),
			Granularity:      "turn",
		})
		index++
	}
	return out, skipped, nil
}

// ParseDir opens ui_messages.json and task_metadata.json under taskDir and parses them
// with ParseTask. task_metadata.json is optional; a missing or unreadable file yields
// no model_usage rather than an error, since Cline task dirs predating that file exist.
func ParseDir(taskDir string) ([]usage.Record, int, error) {
	//nolint:gosec // paths come from local-home discovery globs
	f, err := os.Open(filepath.Join(taskDir, "ui_messages.json"))
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = f.Close() }()

	taskID := filepath.Base(taskDir)
	meta, err := readTaskMetadata(taskDir)
	if err != nil {
		return nil, 0, err
	}
	return ParseTask(f, taskID, meta)
}

func readTaskMetadata(taskDir string) (taskMetadata, error) {
	//nolint:gosec // paths come from local-home discovery globs
	f, err := os.Open(filepath.Join(taskDir, "task_metadata.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return taskMetadata{}, nil
		}
		return taskMetadata{}, err
	}
	defer func() { _ = f.Close() }()

	var meta taskMetadata
	if err := json.NewDecoder(f).Decode(&meta); err != nil {
		return taskMetadata{}, err
	}
	return meta, nil
}

func sortedModelUsage(entries []modelUsageEntry) []modelUsageEntry {
	sorted := make([]modelUsageEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].TS < sorted[j].TS })
	return sorted
}

// modelAt returns the model_id of the latest entry at or before ts, carrying forward
// across requests; if none precede ts it falls back to the earliest known entry.
func modelAt(sorted []modelUsageEntry, ts int64) string {
	if len(sorted) == 0 {
		return ""
	}
	model := sorted[0].ModelID
	for _, e := range sorted {
		if e.TS > ts {
			break
		}
		model = e.ModelID
	}
	return model
}

func millisToTime(ms int64) time.Time {
	return time.UnixMilli(ms).UTC()
}
