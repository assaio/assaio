package docs

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Command is one invocation as a caller types it, without the binary name: "digest",
// "signals coverage". The root is not one of these -- it is the binary, and what it
// contributes is GlobalFlags.
type Command struct {
	Path  string `json:"path"`
	Short string `json:"short"`
	// Hidden commands are exported rather than dropped: a reference that silently omitted one
	// would leave a reader who found it in --help with nowhere to look it up.
	Hidden bool   `json:"hidden"`
	Flags  []Flag `json:"flags"`
}

// TopLevel reports whether this command is a capability of its own rather than a mode of one.
// The distinction is what the published surfaces are held to: a page has to name every
// capability, not every subcommand of one.
func (c Command) TopLevel() bool { return !strings.Contains(c.Path, " ") }

// Flag is one flag declared on its own command -- inherited ones are not repeated, and the
// flags that reach every command are published once as GlobalFlags. Default is the string cobra
// prints in --help, which is what a reader will compare against.
type Flag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type"`
	Default   string `json:"default"`
	Usage     string `json:"usage"`
}

// commands walks the tree and returns it sorted by path. Cobra's own help and completion
// commands are skipped: they are the framework's, not assaio's, and a reference that listed
// them would invite documenting behaviour this repository does not own.
func commands(root *cobra.Command) []Command {
	out := []Command{}
	if root == nil {
		return out
	}
	var walk func(c *cobra.Command, prefix string)
	walk = func(c *cobra.Command, prefix string) {
		for _, sub := range c.Commands() {
			if sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}
			path := strings.TrimSpace(prefix + " " + sub.Name())
			out = append(out, Command{Path: path, Short: sub.Short, Hidden: sub.Hidden, Flags: flags(sub)})
			walk(sub, path)
		}
	}
	walk(root, "")
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// globalFlags are the root's persistent flags, which every command accepts. Listing them once
// here rather than on each command states one fact once instead of thirty-four times.
func globalFlags(root *cobra.Command) []Flag {
	if root == nil {
		return []Flag{}
	}
	out := []Flag{}
	root.PersistentFlags().VisitAll(func(f *pflag.Flag) { out = append(out, newFlag(f)) })
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func flags(c *cobra.Command) []Flag {
	out := []Flag{}
	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		// --help is cobra's own, and it is attached lazily to whichever command runs -- so
		// including it would put a flag on one command and not its siblings, for a reason that
		// has nothing to do with assaio. Skipped for the same reason the help command is.
		if f.Name == "help" {
			return
		}
		out = append(out, newFlag(f))
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func newFlag(f *pflag.Flag) Flag {
	return Flag{
		Name:      f.Name,
		Shorthand: f.Shorthand,
		Type:      f.Value.Type(),
		Default:   f.DefValue,
		Usage:     f.Usage,
	}
}
