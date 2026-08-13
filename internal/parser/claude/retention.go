package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// DefaultRetentionDays is how long Claude Code keeps a transcript when nothing says otherwise.
// This is the number that decides what assaio can ever read: history older than it is deleted by
// the tool, so no backfill can recover it, and a store's own span will sit at it rather than at
// the install date (B156). Anthropic added its own warnings for this in 2.1.217.
const DefaultRetentionDays = 30

// settings is the narrow view of a Claude Code settings file: the one key that decides how much
// history exists to read. Everything else in the file is the user's business and is not decoded.
type settings struct {
	CleanupPeriodDays *int `json:"cleanupPeriodDays"`
}

// Retention is what a settings chain says about how long transcripts survive.
type Retention struct {
	Days int
	// Source is the file the figure came from, "" when no file set it and Days is the default.
	Source string
}

// ReadRetention resolves the effective transcript retention for this machine. Only the two
// machine-wide files are read: the managed policy, which cannot be overridden, and the user's own.
// A project's .claude/settings.json can narrow it further for work inside that repository, and is
// deliberately not consulted here -- doctor answers for the machine, and reporting one repository's
// setting as the machine's would be a claim about history that other repositories do not share.
func ReadRetention(home string) Retention {
	// Managed policy wins over the user's file, which is the order Claude Code applies.
	for _, path := range []string{managedSettingsPath(), filepath.Join(home, ".claude", "settings.json")} {
		if path == "" {
			continue
		}
		if days, ok := cleanupDays(path); ok {
			return Retention{Days: days, Source: path}
		}
	}
	return Retention{Days: DefaultRetentionDays}
}

// cleanupDays reads one settings file. An unreadable or malformed file states nothing rather than
// forcing the default: the caller then goes on to the next file in the chain, which is what Claude
// Code itself does with a file it cannot parse.
func cleanupDays(path string) (int, bool) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is one of two derived locations -- the platform's managed-policy file or $HOME/.claude/settings.json -- never a caller's string.
	if err != nil {
		return 0, false
	}
	var s settings
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, false
	}
	if s.CleanupPeriodDays == nil || *s.CleanupPeriodDays < 0 {
		return 0, false
	}
	return *s.CleanupPeriodDays, true
}

// managedSettingsPath is the enterprise policy file, which exists only on the platforms Claude
// Code documents one for. "" elsewhere, so the chain skips it rather than probing a path that
// cannot exist.
func managedSettingsPath() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/ClaudeCode/managed-settings.json"
	case "linux":
		return "/etc/claude-code/managed-settings.json"
	}
	return ""
}
