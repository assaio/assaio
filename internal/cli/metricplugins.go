package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/plugin"
)

// metricPluginPrefix namespaces every exec metric plugin's validator name; selection,
// listing, and rendering all key off it.
const metricPluginPrefix = "plugin:"

// collectAnalysisResults produces the Results an analyze run renders: every registered
// validator plus every configured metric plugin when names is empty, or exactly the
// named subset -- built-in names and plugin:<name> entries -- in the order given.
func collectAnalysisResults(cmd *cobra.Command, names []string, metricCfgs []config.PluginConfig, in *analyze.Input) ([]analyze.Result, error) {
	if len(names) > 0 {
		return collectNamedResults(cmd, names, metricCfgs, in)
	}
	results := runValidatorResults(analyze.Validators(), in)
	return append(results, runMetricPlugins(cmd.Context(), metricCfgs, in, cmd.ErrOrStderr())...), nil
}

// collectNamedResults resolves explicitly requested names. Unlike the run-everything
// path, a named metric plugin that fails is a hard error: the user asked for exactly
// that result, so a warning-and-continue would silently deliver nothing.
func collectNamedResults(cmd *cobra.Command, names []string, metricCfgs []config.PluginConfig, in *analyze.Input) ([]analyze.Result, error) {
	out := make([]analyze.Result, 0, len(names))
	for _, name := range names {
		if bare, isPlugin := strings.CutPrefix(name, metricPluginPrefix); isPlugin {
			pc, ok := findMetricConfig(metricCfgs, bare)
			if !ok {
				return nil, unknownAnalysisName(name, metricCfgs)
			}
			res, err := runOneMetricPlugin(cmd.Context(), pc, in)
			if err != nil {
				return nil, err
			}
			out = append(out, res)
			continue
		}
		v, ok := analyze.Get(name)
		if !ok {
			return nil, unknownAnalysisName(name, metricCfgs)
		}
		out = append(out, v.Analyze(*in))
	}
	return out, nil
}

func runValidatorResults(validators []analyze.Validator, in *analyze.Input) []analyze.Result {
	out := make([]analyze.Result, len(validators))
	for i, v := range validators {
		out[i] = v.Analyze(*in)
	}
	return out
}

// runMetricPlugins runs every configured metric plugin over in, name-sorted for
// deterministic output. A failing plugin is skipped with one warning line on errW --
// mirroring backfill's plugin Failed handling -- so a broken metric never blocks the
// built-in report.
func runMetricPlugins(ctx context.Context, cfgs []config.PluginConfig, in *analyze.Input, errW io.Writer) []analyze.Result {
	sorted := sortedMetricConfigs(cfgs)
	out := make([]analyze.Result, 0, len(sorted))
	for _, pc := range sorted {
		res, err := runOneMetricPlugin(ctx, pc, in)
		if err != nil {
			_, _ = fmt.Fprintf(errW, "warning: %v\n", err)
			continue
		}
		out = append(out, res)
	}
	return out
}

func runOneMetricPlugin(ctx context.Context, pc config.PluginConfig, in *analyze.Input) (analyze.Result, error) {
	resolved, err := plugin.Resolve(pc)
	if err != nil {
		return analyze.Result{}, fmt.Errorf("metric plugin %s: %w", pc.Name, err)
	}
	return plugin.RunMetric(ctx, resolved, in)
}

func sortedMetricConfigs(cfgs []config.PluginConfig) []config.PluginConfig {
	out := make([]config.PluginConfig, len(cfgs))
	copy(out, cfgs)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func findMetricConfig(cfgs []config.PluginConfig, name string) (config.PluginConfig, bool) {
	for _, pc := range cfgs {
		if pc.Name == name {
			return pc, true
		}
	}
	return config.PluginConfig{}, false
}

// validateAnalysisNames fails fast on an unresolvable requested name, before any store
// is opened or plugin runs -- a typo must error even against an empty store.
func validateAnalysisNames(names []string, metricCfgs []config.PluginConfig) error {
	for _, name := range names {
		if bare, isPlugin := strings.CutPrefix(name, metricPluginPrefix); isPlugin {
			if _, ok := findMetricConfig(metricCfgs, bare); !ok {
				return unknownAnalysisName(name, metricCfgs)
			}
			continue
		}
		if _, ok := analyze.Get(name); !ok {
			return unknownAnalysisName(name, metricCfgs)
		}
	}
	return nil
}

func unknownAnalysisName(name string, metricCfgs []config.PluginConfig) error {
	return fmt.Errorf("unknown validator %q (want one of %s)", name, validAnalysisNames(metricCfgs))
}

// validAnalysisNames renders every runnable name -- registered validators plus
// configured plugin:<name> metrics -- for an unknown-name error message.
func validAnalysisNames(metricCfgs []config.PluginConfig) string {
	all := analyze.Validators()
	names := make([]string, 0, len(all)+len(metricCfgs))
	for _, v := range all {
		names = append(names, v.Name())
	}
	for _, pc := range sortedMetricConfigs(metricCfgs) {
		names = append(names, metricPluginPrefix+pc.Name)
	}
	return strings.Join(names, ", ")
}
