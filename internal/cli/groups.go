package cli

import "github.com/spf13/cobra"

// commandGroups are the --help headings, in the order cobra prints them: the order a
// newcomer meets the tool in, not alphabetical. Titles carry their own colon because cobra
// renders them verbatim next to its own "Additional Commands:".
var commandGroups = []*cobra.Group{
	{ID: "start", Title: "Start here:"},
	{ID: "read", Title: "Read:"},
	{ID: "act", Title: "Act:"},
	{ID: "maintain", Title: "Maintain:"},
	{ID: "team", Title: "Team:"},
	{ID: "extend", Title: "Extend:"},
}

// commandGroupIDs maps a command name to its heading. A command absent from this map is not
// an omission to fix by inventing a heading: cobra prints it under "Additional Commands",
// which is where one that serves another command's workflow rather than a job of its own
// belongs.
var commandGroupIDs = map[string]string{
	"init": "start", "demo": "start", "doctor": "start",

	"report": "read", "analyze": "read", "effectiveness": "read", "status": "read",
	"digest": "read", "explain": "read", "signals": "read",

	"recommend": "act", "reprice": "act", "check": "act", "mark": "act", "share": "act",

	"backfill": "maintain", "compact": "maintain", "clear": "maintain", "reconcile": "maintain",

	"serve": "team", "sync": "team",

	"plugins": "extend", "metrics": "extend", "docs": "extend",
}

// addCommands attaches cmds to root under their --help heading. The groups are declared
// here with the mapping rather than by each command constructor: cobra panics on a GroupID
// its parent was never given, so the two have to be stated in one place.
func addCommands(root *cobra.Command, cmds ...*cobra.Command) {
	root.AddGroup(commandGroups...)
	for _, c := range cmds {
		c.GroupID = commandGroupIDs[c.Name()]
		root.AddCommand(c)
	}
}
