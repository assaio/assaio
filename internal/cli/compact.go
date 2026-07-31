package cli

import (
	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/store"
)

func newCompactCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "compact",
		Short: "Reclaim disk space the store freed but is still holding",
		Long: `Rewrite the store without its free pages and truncate the write-ahead log.

Deleting records -- with clear, or when a vendor rotates away logs assaio had already
read -- frees pages inside the database file without returning them to the filesystem, so
the file stays exactly as large as it was. This is what actually shrinks it.

It is a separate command rather than a step in clear because rewriting needs roughly twice
the store's size in temporary disk space, which no ordinary command should spend
unannounced.`,
		Args: cobra.NoArgs,
		RunE: runCompact,
	}
	addDBFlag(c)
	return c
}

func runCompact(cmd *cobra.Command, _ []string) error {
	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	before, err := st.Size(cmd.Context())
	if err != nil {
		return err
	}
	if err := st.Vacuum(cmd.Context()); err != nil {
		return err
	}
	after, err := st.Size(cmd.Context())
	if err != nil {
		return err
	}
	if after.Bytes >= before.Bytes {
		cmd.Printf("store: %s, nothing to reclaim\n", humanize.Bytes(before.Bytes))
		return nil
	}
	cmd.Printf("store: %s -> %s (reclaimed %s)\n",
		humanize.Bytes(before.Bytes), humanize.Bytes(after.Bytes),
		humanize.Bytes(before.Bytes-after.Bytes))
	return nil
}

// storeSizeLine renders the store's footprint and, when there is space to get back, how.
// Growth stays invisible until someone looks at the file, which is exactly when it is a
// surprise -- so doctor states it on every run.
func storeSizeLine(size store.Size) string {
	if size.Reclaimable == 0 {
		return humanize.Bytes(size.Bytes)
	}
	return humanize.Bytes(size.Bytes) + " · " + humanize.Bytes(size.Reclaimable) +
		" reclaimable — run 'assaio-agent compact'"
}
