package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/recommend"
	"github.com/assaio/assaio/internal/reprice"
)

func newRepriceCmd() *cobra.Command {
	f := repriceFlags{}
	c := &cobra.Command{
		Use:   "reprice",
		Short: "Which plan and which model mix this window's own numbers support",
		Long: `Re-price the window assaio already read against another entry in the same price table.

It answers two questions with one arithmetic: what this same set of turns costs on a different
model, and how its projected monthly rate stands against a flat plan price. Nothing here is a
prediction. A different model does not emit the same tokens for the same request, and assaio has
never seen one do this work -- so every figure states what it holds fixed and what it refuses
to claim, and the experiment it supports names its own rollback and follow-up.

assaio vendors no plan catalogue: a published plan price changes without notice, and a stale
one read as current is a wrong recommendation about real money. A candidate plan is a figure
you read off your vendor's page and pass in.`,
		Example: `  assaio-agent reprice --since 30d
  assaio-agent reprice --plan "Max 5x=100" --plan "Max 20x=200"
  assaio-agent reprice --against claude-haiku-4-5 --format json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReprice(cmd, &f)
		},
	}
	c.Flags().StringVar(&f.since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&f.format, "format", "text", "output format: text|json")
	c.Flags().StringArrayVar(&f.plans, "plan", nil, `candidate flat plan, "name=monthly-price", repeatable`)
	c.Flags().StringArrayVar(&f.against, "against", nil, "extra model to re-price the premium slice onto, repeatable")
	addDBFlag(c)
	return c
}

type repriceFlags struct {
	since, format  string
	plans, against []string
}

func runReprice(cmd *cobra.Command, f *repriceFlags) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("since") {
		f.since = cfg.Since
	}
	start, err := parseSinceAt(f.since, time.Now())
	if err != nil {
		return err
	}
	opts := reprice.Options{Against: f.against}
	for _, spec := range f.plans {
		p, perr := reprice.ParsePlan(spec)
		if perr != nil {
			return perr
		}
		opts.Plans = append(opts.Plans, p)
	}
	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	in, err := buildAnalyzeInput(cmd, st, start)
	if err != nil {
		return err
	}
	window := reprice.Compute(&in, opts)
	// The advice is the one family built on this arithmetic, run over the same Input, so a
	// record quotes figures printed directly above it rather than a second computation of them.
	records := recommend.FromFamily(recommend.FamilyCheaperRoute, &recommend.Evidence{Input: &in})
	return renderReprice(cmd, f.format, &window, records)
}

// repriceDocument is what --format json emits. The arithmetic and the advice it supports ship
// as one document because a consumer that fetched them from two runs could read a
// recommendation against a window that had moved underneath it.
type repriceDocument struct {
	Window          *reprice.Window    `json:"window"`
	Recommendations []recommend.Record `json:"recommendations"`
}

func renderReprice(cmd *cobra.Command, format string, window *reprice.Window, records []recommend.Record) error {
	out := cmd.OutOrStdout()
	switch format {
	case "text":
		if err := reprice.RenderText(out, window); err != nil {
			return err
		}
		return recommend.RenderText(out, records)
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		// An empty list encodes as [], never null: a consumer must not have to tell "assaio
		// abstained" apart from "assaio failed" by the shape of the document.
		if records == nil {
			records = []recommend.Record{}
		}
		return enc.Encode(repriceDocument{Window: window, Recommendations: records})
	default:
		return fmt.Errorf("unknown format %q (want text|json)", format)
	}
}
