package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/runtime"
	"github.com/assaio/assaio/internal/runtime/dcgm"
	"github.com/assaio/assaio/internal/runtime/openmetrics"
	"github.com/assaio/assaio/internal/runtime/vllm"
)

// runtimeSource pairs an adapter's catalog with the two mutually exclusive ways to reach it.
type runtimeSource struct {
	name    string
	catalog []runtime.Capability
	url     string
	file    string
}

func newRuntimeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "runtime",
		Short: "Read a self-hosted inference deployment's own telemetry (experimental)",
		Long: `Read the metrics a self-hosted inference deployment already publishes.

EXPERIMENTAL. This is a feasibility slice, not a monitoring product: one snapshot, read-only,
nothing stored, no cost model and no optimization advice. It exists to find out whether runtime
evidence beside assaio's usage evidence changes a decision anybody makes.

It can only see infrastructure you operate. A hosted Claude, Codex or Gemini session runs on the
vendor's accelerators, which no local signal reveals -- assaio leaves that unknown rather than
estimating it.`,
	}
	c.AddCommand(newRuntimeInspectCmd())
	return c
}

func newRuntimeInspectCmd() *cobra.Command {
	var vllmURL, vllmFile, dcgmURL, dcgmFile, format string
	var timeout time.Duration
	var maxBytes int64
	var maxRedirects int
	c := &cobra.Command{
		Use:   "inspect",
		Short: "Snapshot vLLM and NVIDIA DCGM metric endpoints, and report what they do and do not expose",
		Example: `  assaio-agent runtime inspect --vllm-url http://127.0.0.1:8000/metrics --dcgm-url http://127.0.0.1:9400/metrics
  assaio-agent runtime inspect --vllm-file sample.prom --dcgm-file dcgm.prom`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sources := []runtimeSource{
				{name: vllm.Source, catalog: vllm.Catalog(), url: vllmURL, file: vllmFile},
				{name: dcgm.Source, catalog: dcgm.Catalog(), url: dcgmURL, file: dcgmFile},
			}
			lim := runtime.FetchLimits{Timeout: timeout, MaxBytes: maxBytes, MaxRedirects: maxRedirects}
			return runRuntimeInspect(cmd, sources, lim, format)
		},
	}
	c.Flags().StringVar(&vllmURL, "vllm-url", "", "vLLM metrics endpoint, e.g. http://127.0.0.1:8000/metrics")
	c.Flags().StringVar(&vllmFile, "vllm-file", "", "a saved vLLM exposition to read instead of an endpoint")
	c.Flags().StringVar(&dcgmURL, "dcgm-url", "", "DCGM exporter metrics endpoint, e.g. http://127.0.0.1:9400/metrics")
	c.Flags().StringVar(&dcgmFile, "dcgm-file", "", "a saved DCGM exposition to read instead of an endpoint")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json")
	c.Flags().DurationVar(&timeout, "timeout", runtime.DefaultFetchLimits().Timeout, "bound on one endpoint read")
	c.Flags().Int64Var(&maxBytes, "max-bytes", runtime.DefaultFetchLimits().MaxBytes, "bound on one response body")
	c.Flags().IntVar(&maxRedirects, "max-redirects", runtime.DefaultFetchLimits().MaxRedirects, "how far a redirect chain may lead; 0 forbids redirects")
	return c
}

func runRuntimeInspect(cmd *cobra.Command, sources []runtimeSource, lim runtime.FetchLimits, format string) error {
	requested := make([]runtimeSource, 0, len(sources))
	for _, s := range sources {
		if s.url != "" && s.file != "" {
			return fmt.Errorf("--%s-url and --%s-file are mutually exclusive: pick the live endpoint or the saved snapshot, not both", s.name, s.name)
		}
		if s.url != "" || s.file != "" {
			requested = append(requested, s)
		}
	}
	if len(requested) == 0 {
		return errors.New("nothing to inspect: pass --vllm-url/--vllm-file or --dcgm-url/--dcgm-file")
	}

	now := time.Now()
	snapshots := make([]runtime.Snapshot, 0, len(requested))
	for _, s := range requested {
		snapshots = append(snapshots, inspectSource(cmd.Context(), &s, lim, now))
	}
	return renderRuntime(cmd, snapshots, format)
}

// inspectSource reads one source and never returns an error: an endpoint that could not be
// reached is a finding about the deployment, reported in the same shape as a successful read,
// not a reason to abandon the other source the reader also asked about.
func inspectSource(ctx context.Context, s *runtimeSource, lim runtime.FetchLimits, now time.Time) runtime.Snapshot {
	origin, body, err := readSource(ctx, s, lim)
	if err != nil {
		return runtime.Unreachable(s.name, origin, err.Error(), s.catalog, now)
	}
	doc, err := openmetrics.Parse(bytes.NewReader(body), openmetrics.Limits{})
	if err != nil {
		return runtime.Unreachable(s.name, origin, "the exposition could not be parsed: "+err.Error(), s.catalog, now)
	}
	return runtime.Inspect(s.name, origin, doc, s.catalog, now)
}

func readSource(ctx context.Context, s *runtimeSource, lim runtime.FetchLimits) (origin string, body []byte, err error) {
	if s.file != "" {
		body, err = os.ReadFile(s.file)
		return "file:" + s.file, body, err
	}
	body, err = runtime.Fetch(ctx, s.url, lim)
	return s.url, body, err
}

func renderRuntime(cmd *cobra.Command, snapshots []runtime.Snapshot, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(snapshots)
	case "text":
		for i := range snapshots {
			if err := runtime.RenderText(cmd.OutOrStdout(), &snapshots[i]); err != nil {
				return err
			}
			cmd.Println()
		}
		cmd.Println(runtimeLimitsNote)
		return nil
	default:
		return fmt.Errorf("unknown format %q (want text|json)", format)
	}
}

// runtimeLimitsNote is what one snapshot cannot tell you, printed every run rather than only
// when something looks wrong -- the reading that most needs this sentence is the one that looks
// fine.
const runtimeLimitsNote = `This is one snapshot. It shows what these endpoints publish right now; it cannot show a rate,
a trend, a percentile that means "lately", or a cost. Counters here are cumulative since the
exporter started, and turning one into a throughput needs a second read and the interval
between them. Nothing was stored, and assaio holds no history of any of it.`
