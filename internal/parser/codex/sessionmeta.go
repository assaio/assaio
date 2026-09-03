package codex

import (
	"encoding/json"
	"path/filepath"
	"time"
)

type sessionMeta struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Model     string    `json:"model"`
	Cwd       string    `json:"cwd"`
	// Source is how the run was started, in Codex's own words: "exec" for a scripted
	// `codex exec`, "cli" for the terminal UI, "vscode" for the desktop app. Stored verbatim,
	// the way Claude's entrypoint is, so the scope vocabulary decides what each word means in
	// one place rather than each parser translating on the way in.
	//
	// Raw because the field is a union: a bare string on 2,500 rollouts of the audited corpus
	// and an object -- {"subagent":"review"} -- on 7 of them, where Codex ran the session as a
	// sub-agent of another. Typing it as a string made those 7 session_meta lines fail to
	// unmarshal outright, which cost the whole rollout its session id, project, model and cwd:
	// a field read too confidently is a worse outcome than a field not read at all.
	Source json.RawMessage `json:"source"`
}

// entrypoint is the run's starting point when Codex named one in a word this reading knows how
// to place. The object form is left unstated rather than mapped: those runs are neither a
// person's terminal nor an SDK caller, and the scope that would fit them -- a sub-agent's own
// sequence -- needs a timeline of its own, which this reading does not give them.
func (m *sessionMeta) entrypoint() string {
	var word string
	if err := json.Unmarshal(m.Source, &word); err != nil {
		return ""
	}
	return word
}

func (st *parseState) applySessionMeta(payload json.RawMessage) {
	var m sessionMeta
	if err := json.Unmarshal(payload, &m); err != nil {
		st.skipped++
		return
	}
	st.session = m.ID
	// Parse already advanced st.ts from this line's envelope timestamp; only let the
	// payload override it when the payload actually carries one, so a session_meta whose
	// payload omits the timestamp does not reset st.ts to the zero time.
	if !m.Timestamp.IsZero() {
		st.ts = m.Timestamp
	}
	if st.model == "" {
		st.model = m.Model
	}
	if m.Cwd != "" {
		st.project = filepath.Base(m.Cwd)
		st.cwd = m.Cwd
	}
	if ep := m.entrypoint(); ep != "" {
		st.entrypoint = ep
	}
}
