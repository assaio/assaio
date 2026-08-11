package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/docs"
)

const (
	formatJSON = "json"
	formatHTML = "html"
)

func newDocsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "docs",
		Short: "The enumerable surfaces of this binary, for people and for tools",
		Long: `Everything about assaio that can be enumerated rather than described: the signal
catalog, the source-depth matrix, the validators, the command tree, the config keys, and the
metric-plugin protocol -- read from the live registries of the binary you are running.

The published reference and the project's website are generated from this, and checked against
it, so a page cannot describe a capability the binary does not have. What a figure *means* stays
prose in docs/; only what can be listed is generated.`,
	}
	c.AddCommand(newDocsExportCmd())
	return c
}

func newDocsExportCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "export",
		Short: "Write the reference as JSON, or as a self-contained HTML page",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			format, _ := cmd.Flags().GetString("format")
			ref := docs.Export(cmd.Root())
			switch format {
			case formatJSON:
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(ref)
			case formatHTML:
				// No guide list: a reader running this has the binary, not the repository.
				// The published site passes its own, which is what puts the reference inside
				// the documentation rather than beside it.
				_, err := cmd.OutOrStdout().Write(docs.HTML(ref, nil))
				return err
			default:
				return fmt.Errorf("unknown format %q (want json or html)", format)
			}
		},
	}
	c.Flags().String("format", formatJSON, "output format: json|html")
	return c
}
