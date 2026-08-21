package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/recommend"
)

func newRecommendCmd() *cobra.Command {
	var since, format string
	c := &cobra.Command{
		Use:   "recommend",
		Short: "Experiments this window's evidence supports, each with its rollback and follow-up",
		Long: `Derive the small number of experiments this window's evidence actually supports.

Each one states what triggered it, what it requires, what it risks, how to undo it, when to
look again, and the figure that would show whether it worked -- so it can be judged before it
is run and checked after. assaio has no counterfactual and predicts no number.

Abstention is the default: a window whose evidence is thin, whose verdicts are low-confidence,
or whose metrics are missing a declared input produces nothing at all rather than a plausible
suggestion.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRecommend(cmd, &since, &format)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json")
	addDBFlag(c)
	return c
}

func runRecommend(cmd *cobra.Command, since, format *string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("since") {
		*since = cfg.Since
	}
	start, err := parseSinceAt(*since, time.Now())
	if err != nil {
		return err
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
	// The verdicts come from the same evaluation `analyze` prints, so a recommendation quotes a
	// figure the reader can go and see rather than one recomputed behind their back.
	results := make([]analyze.Result, 0, len(analyze.Validators()))
	for _, v := range analyze.Validators() {
		results = append(results, analyze.Evaluate(v, &in))
	}
	records := recommend.From(&recommend.Evidence{Input: &in, Results: results})

	switch *format {
	case "text":
		return recommend.RenderText(cmd.OutOrStdout(), records)
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		// An empty list encodes as [], never null: a consumer must not have to tell "assaio
		// abstained" apart from "assaio failed" by the shape of the document.
		if records == nil {
			records = []recommend.Record{}
		}
		return enc.Encode(records)
	default:
		return fmt.Errorf("unknown format %q (want text|json)", *format)
	}
}
