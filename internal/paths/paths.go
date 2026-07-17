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
// the VS Code extension's global storage, and the Cline CLI data directory. Each root's
// tasks live under a "tasks" subdirectory.
func ClineRoots(home string) []string {
	return clineRootsFor(runtime.GOOS, home, os.Getenv("APPDATA"))
}

// clineRootsFor is the pure, OS-parameterized implementation behind ClineRoots.
func clineRootsFor(goos, home, appdata string) []string {
	const extensionID = "saoudrizwan.claude-dev"
	var vscodeGlobalStorage string
	switch goos {
	case "darwin":
		vscodeGlobalStorage = filepath.Join(home, "Library", "Application Support", "Code", "User", "globalStorage")
	case "windows":
		if appdata == "" {
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		vscodeGlobalStorage = filepath.Join(appdata, "Code", "User", "globalStorage")
	default:
		vscodeGlobalStorage = filepath.Join(home, ".config", "Code", "User", "globalStorage")
	}
	return []string{
		filepath.Join(vscodeGlobalStorage, extensionID),
		filepath.Join(home, ".cline", "data"),
	}
}
