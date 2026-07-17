package cli

import (
	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(version.Version)
			return nil
		},
	}
}
