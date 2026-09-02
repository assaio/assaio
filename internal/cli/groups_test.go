package cli

import (
	"bytes"
	"strings"
	"testing"
)

// TestCommandGroupIDsNameRealCommands guards the silent half of the mapping: a renamed or
// misspelled command key does not fail, it just drops that command back into "Additional
// Commands", where nobody notices it is missing from its heading.
func TestCommandGroupIDsNameRealCommands(t *testing.T) {
	registered := map[string]bool{}
	for _, c := range NewRootCmd().Commands() {
		registered[c.Name()] = true
	}
	for name := range commandGroupIDs {
		if !registered[name] {
			t.Errorf("commandGroupIDs names %q, which is not a registered command", name)
		}
	}
}

func TestCommandGroupIDsResolveToDeclaredGroups(t *testing.T) {
	declared := map[string]bool{}
	for _, g := range commandGroups {
		declared[g.ID] = true
	}
	for name, id := range commandGroupIDs {
		if !declared[id] {
			t.Errorf("command %q is in group %q, which no group declares", name, id)
		}
	}
}

// TestHelpGroupsEveryCommand walks the rendered help, so the assertion is about what a
// reader sees rather than about the map that produced it.
func TestHelpGroupsEveryCommand(t *testing.T) {
	help := renderHelp(t)
	titles := map[string]string{}
	for _, g := range commandGroups {
		if !strings.Contains(help, g.Title) {
			t.Errorf("--help never prints the %q heading:\n%s", g.Title, help)
		}
		titles[g.ID] = g.Title
	}
	if !strings.Contains(help, additionalCommands) {
		t.Errorf("ungrouped commands must still be listed:\n%s", help)
	}
	for name, id := range commandGroupIDs {
		if got := headingOf(help, name, titles); got != titles[id] {
			t.Errorf("%q prints under %q, want %q", name, got, titles[id])
		}
	}
}

// additionalCommands is cobra's own heading for a command in no group.
const additionalCommands = "Additional Commands:"

func renderHelp(t *testing.T) string {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// headingOf returns the last heading printed before the line listing command name.
func headingOf(help, name string, titles map[string]string) string {
	headings := map[string]bool{additionalCommands: true}
	for _, title := range titles {
		headings[title] = true
	}
	heading := ""
	for _, line := range strings.Split(help, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case headings[trimmed]:
			heading = trimmed
		case strings.HasPrefix(trimmed, name+" "):
			return heading
		}
	}
	return ""
}
