package cli

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/signal"
)

func newSignalsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "signals",
		Short: "What assaio can report, what each figure means, and what your data supports",
		Long: `A source-depth matrix answers "what can this tool tell me". This answers the
question people actually have: can it tell me *this*, on this machine, from the data I have.

'list' names every signal. 'describe' says what one means -- including what a zero means,
which is the difference between "no rework happened" and "this source never recorded rework".
'coverage' reads your own store and reports, per signal, the share of tokens coming from
sources that can answer it.`,
	}
	c.AddCommand(newSignalsListCmd(), newSignalsDescribeCmd(), newSignalsCoverageCmd())
	return c
}

func newSignalsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List every signal with its unit, status and the source capability it needs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			catalog := signal.Catalog()
			for i := range catalog {
				s := &catalog[i]
				cmd.Printf("%-26s %-8s %-9s %s\n", s.ID, s.Unit, s.Status, sourcesFor(s.ID))
			}
			cmd.Printf("\n%d signals. 'signals describe <id>' for what one means.\n", len(signal.Catalog()))
			return nil
		},
	}
}

func newSignalsDescribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe <id>",
		Short: "Explain one signal: what it counts, where it is honest, and what a zero means",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, ok := signal.Get(args[0])
			if !ok {
				return fmt.Errorf("unknown signal %q (see 'signals list')", args[0])
			}
			cmd.Printf("%s -- %s\n\n", s.ID, s.Title)
			cmd.Printf("  %s\n\n", s.Describe)
			cmd.Printf("  unit:        %s\n", s.Unit)
			cmd.Printf("  status:      %s\n", s.Status)
			cmd.Printf("  answered by: %s\n", sourcesFor(s.ID))
			cmd.Printf("  read at:     %s\n", strings.Join(s.Grains, ", "))
			cmd.Printf("  a zero means: %s\n", s.ZeroMeans)
			return nil
		},
	}
}

func newSignalsCoverageCmd() *cobra.Command {
	var since string
	c := &cobra.Command{
		Use:   "coverage",
		Short: "Which signals your own data can actually support",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return runSignalsCoverage(cmd, &since) },
	}
	c.Flags().StringVar(&since, "since", "30d", "window to read, e.g. 7d")
	addDBFlag(c)
	return c
}

func runSignalsCoverage(cmd *cobra.Command, since *string) error {
	if err := resolveSince(cmd, since); err != nil {
		return err
	}
	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	start, err := parseSinceAt(*since, time.Now())
	if err != nil {
		return err
	}
	rows, err := st.Usage(cmd.Context(), start)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return emptyStatusHint(cmd)
	}
	renderSignalSupport(cmd, signal.Coverage(analyze.TokensByTool(rows)), *since)
	return nil
}

// renderSignalSupport groups by verdict rather than listing every signal flat: what a reader
// wants first is the short list of things their data cannot answer at all.
func renderSignalSupport(cmd *cobra.Command, support []signal.Support, since string) {
	byVerdict := map[string][]signal.Support{}
	for i := range support {
		byVerdict[support[i].Verdict] = append(byVerdict[support[i].Verdict], support[i])
	}
	cmd.Printf("signal coverage · last %s · %d signals\n\n", since, len(support))
	for _, v := range []struct{ key, heading string }{
		{signal.Full, "fully supported -- every token in the window can answer these"},
		{signal.Partial, "partially supported -- real, but describing less than the whole window"},
		{signal.None, "not supported -- no source in this window records them"},
	} {
		group := byVerdict[v.key]
		if len(group) == 0 {
			continue
		}
		cmd.Printf("%s (%d)\n", v.heading, len(group))
		for i := range group {
			cmd.Printf("  %-26s %s\n", group[i].Signal.ID, supportDetail(&group[i]))
		}
		cmd.Println()
	}
}

// sourcesFor names the in-tree parsers that produce a signal, which is what makes the
// catalog readable without a store: "no source records this" is an answer too.
func sourcesFor(id string) string {
	var out []string
	for _, d := range parser.Depths() {
		if slices.Contains(d.Answers, id) {
			out = append(out, d.Tool)
		}
	}
	if len(out) == 0 {
		return "no in-tree source"
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func supportDetail(s *signal.Support) string {
	if s.Verdict == signal.None {
		return s.Signal.ZeroMeans
	}
	sources := s.Sources
	sort.Strings(sources)
	return fmt.Sprintf("%s of tokens · %s", analyze.HonestPercent(s.Share), strings.Join(sources, ", "))
}
