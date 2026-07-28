package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/i18n"
)

func newExplainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain [validator]",
		Short: "Print a metric's long-form page: what it measures, how to read it, what to do about it",
		Long: `Print the long-form page for one metric: what it measures, how to read it, what to do
about it, and the limits that keep it honest. With no argument, list every metric.

This is documentation, not measurement -- it never opens the store, so it works before
any data has been imported. To see the metric against your own usage, run
'assaio-agent analyze <validator>'.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runExplainList(cmd)
			}
			return runExplain(cmd, args[0])
		},
	}
}

// runExplainList prints every metric that has a page, so the argument to pass is
// discoverable without leaving the command.
func runExplainList(cmd *cobra.Command) error {
	for _, v := range analyze.Validators() {
		cmd.Printf("%-19s %s\n", v.Name(), v.Describe())
	}
	return nil
}

func runExplain(cmd *cobra.Command, name string) error {
	page, ok := i18n.Explain(name)
	if !ok {
		return fmt.Errorf("unknown metric %q; known metrics: %s", name, strings.Join(validatorNames(), ", "))
	}
	cmd.Println(page)
	return nil
}

func validatorNames() []string {
	validators := analyze.Validators()
	names := make([]string, 0, len(validators))
	for _, v := range validators {
		names = append(names, v.Name())
	}
	return names
}
