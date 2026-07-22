// Package paths resolves assaio's data, config, and tool-log filesystem locations.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// Home returns the current user's home directory.
func Home() (string, error) { return os.UserHomeDir() }

// DataDir returns the assaio data directory, honoring XDG_DATA_HOME.
func DataDir() (string, error) {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "assaio"), nil
	}
	home, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "assaio"), nil
}

// DBPath returns the path to the assaio SQLite database.
func DBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "assaio.db"), nil
}

// ConfigPath returns the path to the assaio config file, honoring XDG_CONFIG_HOME.
func ConfigPath() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "assaio", "config.yaml"), nil
	}
	home, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "assaio", "config.yaml"), nil
}

// ClaudeRoot returns the root directory of Claude Code session logs under home.
func ClaudeRoot(home string) string {
	return filepath.Join(home, ".claude", "projects")
}

// CodexRoots returns the root directories of Codex session logs under home.
func CodexRoots(home string) []string {
	base := filepath.Join(home, ".codex")
	return []string{filepath.Join(base, "sessions"), filepath.Join(base, "archived_sessions")}
}

// GeminiRoot returns the root directory of Gemini CLI chat logs under home.
func GeminiRoot(home string) string {
	return filepath.Join(home, ".gemini")
}

// ClineRoots returns the root directories that may contain Cline task data under home:
// the Cline extension's global storage in each supported VS Code-family editor, and the
// Cline CLI data directory. Each root's tasks live under a "tasks" subdirectory.
func ClineRoots(home string) []string {
	return clineRootsFor(runtime.GOOS, home, os.Getenv("APPDATA"))
}

// clineEditorDirs are the VS Code-family user-data directory names that can host the Cline
// extension's globalStorage: stable VS Code, Insiders, VSCodium, and Cursor. The same
// publisher id lives under each, so Cline data is found wherever the user runs it.
var clineEditorDirs = []string{"Code", "Code - Insiders", "VSCodium", "Cursor"}

// clineRootsFor is the pure, OS-parameterized implementation behind ClineRoots: one
// globalStorage root per editor in clineEditorDirs order, then the Cline CLI data
// directory. Non-existent roots are skipped harmlessly at discovery time.
func clineRootsFor(goos, home, appdata string) []string {
	const extensionID = "saoudrizwan.claude-dev"
	var userData string
	switch goos {
	case "darwin":
		userData = filepath.Join(home, "Library", "Application Support")
	case "windows":
		if appdata == "" {
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		userData = appdata
	default:
		userData = filepath.Join(home, ".config")
	}
	roots := make([]string, 0, len(clineEditorDirs)+1)
	for _, editor := range clineEditorDirs {
		roots = append(roots, filepath.Join(userData, editor, "User", "globalStorage", extensionID))
	}
	return append(roots, filepath.Join(home, ".cline", "data"))
}
