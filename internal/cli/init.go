package cli

import (
	"bufio"
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/ingest"
	"github.com/assaio/assaio/internal/paths"
)

// initSourceHint is what a machine with no detected tool needs: the exact shape of the
// config key that points assaio somewhere else, rather than a bare "nothing found".
const initSourceHint = `No supported tool's logs were found in the default locations.

If your logs live elsewhere, point assaio at them in config.yaml:

sources:
  claude: [/path/to/.claude/projects]
  codex:  [/path/to/.codex/sessions]

Run 'assaio-agent doctor' to see the locations that were checked, or 'assaio-agent demo'
to see a full report on bundled sample data first.`

func newInitCmd() *cobra.Command {
	var nonInteractive bool
	c := &cobra.Command{
		Use:   "init",
		Short: "Set up a first run: show what will be read, import it, write the report",
		Long: `Walk a first run end to end: detect which supported tools have logs on this machine,
print exactly which directories will be read before reading anything, import them, and
write the offline Assay report.

Nothing leaves the machine and nothing is written outside assaio's own data directory and
the report file. A config file is written only if one is actually needed -- with the
default log locations there is nothing to configure, and a file that restates the defaults
is one more thing to maintain.

--non-interactive skips the confirmation, for packaging smoke tests and scripted setup.

There is deliberately no --db: init imports through backfill, which only ever writes this
machine's own store, so pointing the report half elsewhere would have shown an empty first
run against a database init never filled.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd, nonInteractive)
		},
	}
	c.Flags().BoolVar(&nonInteractive, "non-interactive", false, "skip the confirmation prompt")
	return c
}

func runInit(cmd *cobra.Command, nonInteractive bool) error {
	home, err := paths.Home()
	if err != nil {
		return err
	}
	cfg, err := loadConfigLenient(cmd)
	if err != nil {
		return err
	}
	scans := scanSources(home, &cfg.Sources)

	cmd.Println("assaio reads local session logs. On this machine it found:")
	cmd.Println()
	found := 0
	for i := range scans {
		if scans[i].found == 0 {
			continue
		}
		found += scans[i].found
		cmd.Printf("  %-12s %s\n", scans[i].tool, strings.Join(scans[i].roots, ", "))
		cmd.Printf("  %-12s %s\n", "", scans[i].activity)
	}
	if found == 0 {
		cmd.Println(initSourceHint)
		return nil
	}
	cmd.Println()
	cmd.Println("Only token counts and activity counts are extracted -- never prompts, model")
	cmd.Println("output, or the content of any file. See PRIVACY.md.")
	cmd.Println()

	if !nonInteractive && !confirmed(cmd) {
		cmd.Println("Nothing was read.")
		return nil
	}
	return importAndReport(cmd)
}

// confirmed asks once before anything is read. A first run is the moment a user most wants
// to know what a tool is about to touch, so the default is to stop.
func confirmed(cmd *cobra.Command) bool {
	cmd.Print("Read these and build a report? [y/N] ")
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

// importAndReport runs the two commands a first run would otherwise have to discover, and
// ends on the one thing a new user can act on: a file to open.
func importAndReport(cmd *cobra.Command) error {
	if err := runBackfill(cmd, ingest.Options{}); err != nil {
		return err
	}
	cmd.Println()
	cmd.Println("imported.")
	since, output := "30d", dashboardDefaultOutput
	if err := runDashboard(cmd, &since, &output); err != nil {
		return err
	}
	cmd.Println()
	cmd.Printf("Next: open %s\n", output)
	cmd.Println("Then: 'assaio-agent status' for the terminal overview, 'assaio-agent analyze'")
	cmd.Println("for the metric verdicts, and 'assaio-agent doctor' for what assaio can and")
	cmd.Println("cannot see.")
	return nil
}
