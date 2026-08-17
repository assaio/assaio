package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/ingest"
	"github.com/assaio/assaio/internal/share"
	"github.com/assaio/assaio/internal/store"
)

// shareDefaultOutput is where the preview is written when --output is not given.
const shareDefaultOutput = "assaio-share.html"

// shareFirstRunNotice is what an empty store sees before anything is read. `share` is the
// command a first-time user is most likely to run before any other, so it imports rather
// than sending them away to `init` -- but it still says what it is about to read first,
// which is the promise every other entry point keeps.
const shareFirstRunNotice = `No usage stored yet -- reading this machine's existing AI session logs first.
Only token and activity counts are extracted, never prompts or file contents, and nothing
leaves this machine.`

func newShareCmd() *cobra.Command {
	var since, output, format string
	var demo, noOpen bool
	c := &cobra.Command{
		Use:   "share",
		Short: "Build a postable card, reel and text about this machine's AI usage",
		Long: `Render one window as something fit to post in public: a square reel that walks through
the numbers, the archetype and the session fingerprint, plus the still frame and the post
text beside it.

Run it with no flags. An empty store imports the local logs first, so a fresh install
reaches a finished card in one command with nothing to configure.

Redaction is the feature, not a flag: no repository, member, path, branch, skill or
sub-agent name can reach the card. Tools and models are named, because which coding agent
and which model ran is a public fact about a vendor rather than about a person. The
preview is a local file that makes no network request; posting is your own paste.

There is deliberately no --db, for the reason init has none: share imports through backfill,
which only ever writes this machine's own store. Accepting the flag would have imported here
and reported from there -- and the import prunes trace steps past the horizon, which no later
backfill can recover once the source has deleted its transcripts.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runShare(cmd, shareOpts{since: since, output: output, format: format, demo: demo, noOpen: noOpen})
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&output, "output", shareDefaultOutput, "preview output path")
	c.Flags().StringVar(&format, "format", "html", "output format: html|text")
	c.Flags().BoolVar(&demo, "demo", false, "render bundled sample data instead of this machine's usage")
	c.Flags().BoolVar(&noOpen, "no-open", false, "write the preview without opening a browser")
	return c
}

type shareOpts struct {
	since, output, format string
	demo, noOpen          bool
}

func runShare(cmd *cobra.Command, o shareOpts) error {
	if o.format != "html" && o.format != "text" {
		return fmt.Errorf("unknown format %q (want html|text)", o.format)
	}
	if o.demo {
		return shareDemo(cmd, o)
	}
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("since") {
		o.since = cfg.Since
	}
	start, err := parseSinceAt(o.since, time.Now())
	if err != nil {
		return err
	}
	st, empty, err := shareStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	if empty {
		cmd.Println(initSourceHint)
		return nil
	}
	return shareFrom(cmd, st, start, o, false)
}

// shareStore opens the report store and, when it holds nothing, imports the machine's
// logs before handing it back. empty is true only when there was still no usage after
// that import -- a machine with no supported tool, which gets the same hint `init` gives
// rather than a card built on nothing.
func shareStore(cmd *cobra.Command) (*store.Store, bool, error) {
	st, err := openReportStore(cmd)
	if err != nil {
		return nil, false, err
	}
	n, err := st.Count(cmd.Context())
	if err != nil {
		_ = st.Close()
		return nil, false, err
	}
	if n > 0 {
		return st, false, nil
	}
	_ = st.Close()
	cmd.Println(shareFirstRunNotice)
	cmd.Println()
	if err := runBackfill(cmd, ingest.Options{}); err != nil {
		return nil, false, err
	}
	cmd.Println()
	if st, err = openReportStore(cmd); err != nil {
		return nil, false, err
	}
	if n, err = st.Count(cmd.Context()); err != nil {
		_ = st.Close()
		return nil, false, err
	}
	return st, n == 0, nil
}

// shareDemo renders the bundled sample usage in a throwaway store, the same fixture
// `demo` reports on. The card says it is sample data on its own first frame, so a shared
// image can never be mistaken for a real machine's numbers.
func shareDemo(cmd *cobra.Command, o shareOpts) error {
	dir, err := os.MkdirTemp("", "assaio-share-demo")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	st, err := store.Open(filepath.Join(dir, "demo.db"))
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	now := time.Now()
	if _, err := st.Insert(cmd.Context(), demoRecords(now)); err != nil {
		return err
	}
	// The fixture spans demoWindowDays whatever --since says, so the card is labelled from
	// what was read. A header reading "last 7 days · 20 active days" refutes itself.
	o.since = fmt.Sprintf("%dd", demoWindowDays)
	return shareFrom(cmd, st, now.AddDate(0, 0, -demoWindowDays), o, true)
}

func shareFrom(cmd *cobra.Command, st *store.Store, start time.Time, o shareOpts, sample bool) error {
	in, err := buildAnalyzeInput(cmd, st, start)
	if err != nil {
		return err
	}
	// The store can hold plenty and this window still hold nothing -- share --since 1d after an
	// idle day. Saying so beats handing over a card with no figures on it, which is what the
	// renderer would otherwise draw.
	if len(in.Usage) == 0 {
		cmd.Printf("No usage in the %s window, so there is nothing to render. Try a wider --since.\n", windowLabel(o.since))
		return nil
	}
	assay := share.Build(in, runValidatorResults(analyze.Validators(), &in), windowLabel(o.since), sample)
	if o.format == "text" {
		cmd.Print(share.Text(&assay))
		return nil
	}
	if err := writeShareFile(o.output, &assay); err != nil {
		return err
	}
	cmd.Printf("Wrote %s -- preview it, then copy the frame or save the reel.\n", o.output)
	if o.noOpen {
		return nil
	}
	if err := openInBrowser(cmd.Context(), o.output); err != nil {
		cmd.Printf("Could not open a browser (%v); open %s yourself.\n", err, o.output)
	}
	return nil
}

// writeShareFile creates (or truncates) path. The preview is a shareable artifact, so its
// perms are intentionally world-readable, matching the dashboard's.
func writeShareFile(path string, a *share.Assay) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644) //nolint:gosec // shareable HTML artifact, deliberately world-readable
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return share.RenderHTML(f, a)
}
