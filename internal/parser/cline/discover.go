package cline

import "path/filepath"

// Discover returns task directories under one Cline root (VS Code extension storage or
// the Cline CLI data dir). A task directory is a child of <root>/tasks that contains
// ui_messages.json.
func Discover(root string) ([]string, error) {
	found, err := filepath.Glob(filepath.Join(root, "tasks", "*", "ui_messages.json"))
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, f := range found {
		dirs = append(dirs, filepath.Dir(f))
	}
	return dirs, nil
}
