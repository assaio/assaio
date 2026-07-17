package paths

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestHomeErrorsWhenUnset(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME is not the lookup var on windows")
	}
	t.Setenv("HOME", "")
	if _, err := Home(); err == nil {
		t.Fatal("Home() err = nil, want an error when $HOME is unset")
	}
}

func TestDataDir(t *testing.T) {
	xdgDataHome := filepath.Join(string(filepath.Separator), "xdg", "data")
	home := filepath.Join(string(filepath.Separator), "home", "dev")
	tests := []struct {
		name        string
		xdgDataHome string
		home        string
		want        string
	}{
		{
			name:        "XDG_DATA_HOME set",
			xdgDataHome: xdgDataHome,
			home:        home,
			want:        filepath.Join(xdgDataHome, "assaio"),
		},
		{
			name:        "XDG_DATA_HOME unset falls back to home",
			xdgDataHome: "",
			home:        home,
			want:        filepath.Join(home, ".local", "share", "assaio"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", tt.xdgDataHome)
			t.Setenv("HOME", tt.home)
			got, err := DataDir()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("DataDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDBPath(t *testing.T) {
	xdgDataHome := filepath.Join(string(filepath.Separator), "xdg", "data")
	home := filepath.Join(string(filepath.Separator), "home", "dev")
	tests := []struct {
		name        string
		xdgDataHome string
		home        string
		want        string
	}{
		{
			name:        "XDG_DATA_HOME set",
			xdgDataHome: xdgDataHome,
			home:        home,
			want:        filepath.Join(xdgDataHome, "assaio", "assaio.db"),
		},
		{
			name:        "XDG_DATA_HOME unset falls back to home",
			xdgDataHome: "",
			home:        home,
			want:        filepath.Join(home, ".local", "share", "assaio", "assaio.db"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", tt.xdgDataHome)
			t.Setenv("HOME", tt.home)
			got, err := DBPath()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("DBPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigPath(t *testing.T) {
	xdgConfigHome := filepath.Join(string(filepath.Separator), "xdg", "config")
	home := filepath.Join(string(filepath.Separator), "home", "dev")
	tests := []struct {
		name          string
		xdgConfigHome string
		home          string
		want          string
	}{
		{
			name:          "XDG_CONFIG_HOME set",
			xdgConfigHome: xdgConfigHome,
			home:          home,
			want:          filepath.Join(xdgConfigHome, "assaio", "config.yaml"),
		},
		{
			name:          "XDG_CONFIG_HOME unset falls back to home",
			xdgConfigHome: "",
			home:          home,
			want:          filepath.Join(home, ".config", "assaio", "config.yaml"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome)
			t.Setenv("HOME", tt.home)
			got, err := ConfigPath()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("ConfigPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDBPathPropagatesHomeError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME is not the lookup var on windows")
	}
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "")
	if _, err := DBPath(); err == nil {
		t.Fatal("DBPath() err = nil, want an error when home lookup fails")
	}
}

func TestDataDirPropagatesHomeError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME is not the lookup var on windows")
	}
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "")
	if _, err := DataDir(); err == nil {
		t.Fatal("DataDir() err = nil, want an error when home lookup fails")
	}
}

func TestConfigPathPropagatesHomeError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME is not the lookup var on windows")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if _, err := ConfigPath(); err == nil {
		t.Fatal("ConfigPath() err = nil, want an error when home lookup fails")
	}
}

func TestRoots(t *testing.T) {
	home := "/home/dev"
	if got := ClaudeRoot(home); got != filepath.Join(home, ".claude", "projects") {
		t.Fatalf("ClaudeRoot = %q", got)
	}
	roots := CodexRoots(home)
	want := []string{
		filepath.Join(home, ".codex", "sessions"),
		filepath.Join(home, ".codex", "archived_sessions"),
	}
	if len(roots) != 2 || roots[0] != want[0] || roots[1] != want[1] {
		t.Fatalf("CodexRoots = %v", roots)
	}
	if got := GeminiRoot(home); got != filepath.Join(home, ".gemini") {
		t.Fatalf("GeminiRoot = %q", got)
	}
}

func TestClineRoots(t *testing.T) {
	home := "/home/dev"
	roots := ClineRoots(home)
	if len(roots) != 2 {
		t.Fatalf("ClineRoots = %v, want 2 entries", roots)
	}
	if roots[1] != filepath.Join(home, ".cline", "data") {
		t.Fatalf("ClineRoots[1] = %q", roots[1])
	}
	if filepath.Base(roots[0]) != "saoudrizwan.claude-dev" {
		t.Fatalf("ClineRoots[0] = %q, want to end in extension id", roots[0])
	}
}

func TestClineRootsFor(t *testing.T) {
	home := filepath.FromSlash("/home/dev")
	tests := []struct {
		name    string
		goos    string
		appdata string
		want    string
	}{
		{
			name: "darwin",
			goos: "darwin",
			want: filepath.Join(home, "Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev"),
		},
		{
			name: "linux",
			goos: "linux",
			want: filepath.Join(home, ".config", "Code", "User", "globalStorage", "saoudrizwan.claude-dev"),
		},
		{
			name:    "windows with APPDATA",
			goos:    "windows",
			appdata: filepath.FromSlash(`C:/Users/dev/AppData/Roaming`),
			want:    filepath.Join(filepath.FromSlash("C:/Users/dev/AppData/Roaming"), "Code", "User", "globalStorage", "saoudrizwan.claude-dev"),
		},
		{
			name: "windows without APPDATA falls back to home",
			goos: "windows",
			want: filepath.Join(home, "AppData", "Roaming", "Code", "User", "globalStorage", "saoudrizwan.claude-dev"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roots := clineRootsFor(tt.goos, home, tt.appdata)
			if len(roots) != 2 {
				t.Fatalf("clineRootsFor(%q) = %v, want 2 entries", tt.goos, roots)
			}
			if roots[0] != tt.want {
				t.Fatalf("clineRootsFor(%q)[0] = %q, want %q", tt.goos, roots[0], tt.want)
			}
			if roots[1] != filepath.Join(home, ".cline", "data") {
				t.Fatalf("clineRootsFor(%q)[1] = %q", tt.goos, roots[1])
			}
		})
	}
}
