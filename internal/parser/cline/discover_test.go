package cline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverFindsTaskDirs(t *testing.T) {
	root := t.TempDir()

	task1 := filepath.Join(root, "tasks", "task1")
	task2 := filepath.Join(root, "tasks", "task2")
	if err := os.MkdirAll(task1, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(task2, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(task1, "ui_messages.json"), []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(task2, "ui_messages.json"), []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A task-shaped directory without ui_messages.json must not be discovered.
	if err := os.MkdirAll(filepath.Join(root, "tasks", "empty"), 0o750); err != nil {
		t.Fatal(err)
	}

	dirs, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 2 {
		t.Fatalf("got %d dirs want 2: %v", len(dirs), dirs)
	}
}

func TestDiscoverMissingRoot(t *testing.T) {
	dirs, err := Discover(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 0 {
		t.Fatalf("got %d dirs want 0", len(dirs))
	}
}
