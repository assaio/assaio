package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/plugin"
	"github.com/assaio/assaio/internal/store"
)

// gateOnRules runs the configured rule plugins over this window's verdicts, prints their
// alerts, and returns what must fail the gate. Nothing runs -- and nothing is printed --
// when no rules are configured, so check's cost is unchanged for everyone else.
func gateOnRules(cmd *cobra.Command, cfg *config.Config, st *store.Store, start time.Time) ([]string, error) {
	if len(cfg.Rules) == 0 {
		return nil, nil
	}
	verdicts, missing, err := windowVerdicts(cmd, cfg, st, start)
	if err != nil {
		return nil, err
	}
	alerts, unevaluated := runRulePlugins(cmd.Context(), cfg.Rules, verdicts, cmd.ErrOrStderr())
	// A metric plugin that failed leaves a hole in the verdict set the rules just read, so
	// its absence fails the gate exactly like a rule that did not run: a rule cannot raise
	// an alert about a verdict it was never shown.
	notRun := append(kinded("metric plugin", missing), kinded("rule plugin", unevaluated)...)
	if err := renderAlerts(cmd, alerts, notRun); err != nil {
		return nil, err
	}
	failures := make([]string, 0, len(notRun))
	for _, what := range notRun {
		failures = append(failures, what+" did not run")
	}
	return append(failures, gatingAlerts(alerts)...), nil
}

// kinded prefixes each plugin name with what it is, so a gate failure says which kind of
// plugin left the hole rather than blaming a rule for a metric's absence.
func kinded(kind string, names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, kind+" "+name)
	}
	return out
}

// windowVerdicts computes the same verdict set `analyze` renders -- every registered
// validator plus every configured metric plugin -- so a rule can gate on a company's own
// exec metric, not just on the built-ins. missing names the metric plugins that failed:
// their verdicts are absent, and a gate must not pass on a set it knows is incomplete.
func windowVerdicts(cmd *cobra.Command, cfg *config.Config, st *store.Store, start time.Time) (verdicts []analyze.Result, missing []string, err error) {
	in, err := buildAnalyzeInput(cmd, st, start)
	if err != nil {
		return nil, nil, err
	}
	verdicts = runValidatorResults(analyze.Validators(), &in)
	plugins, failed := runMetricPlugins(cmd.Context(), cfg.Metrics, &in, cmd.ErrOrStderr())
	return append(verdicts, plugins...), failed, nil
}

// runRulePlugins runs every configured rule plugin over verdicts, name-sorted for
// deterministic output. A plugin that fails its protocol or contract has its alerts
// dropped whole and is reported on errW and as an unevaluated rule: the remaining rules
// still run, but the gate must not report a pass it never established.
func runRulePlugins(ctx context.Context, cfgs []config.PluginConfig, verdicts []analyze.Result, errW io.Writer) (alerts []plugin.Alert, unevaluated []string) {
	for _, pc := range sortedPluginConfigs(cfgs) {
		raised, err := runOneRulePlugin(ctx, pc, verdicts)
		if err != nil {
			_, _ = fmt.Fprintf(errW, "warning: %v\n", err)
			unevaluated = append(unevaluated, pc.Name)
			continue
		}
		alerts = append(alerts, raised...)
	}
	return alerts, unevaluated
}

func runOneRulePlugin(ctx context.Context, pc config.PluginConfig, verdicts []analyze.Result) ([]plugin.Alert, error) {
	resolved, err := plugin.Resolve(pc)
	if err != nil {
		return nil, fmt.Errorf("rule plugin %s: %w", pc.Name, err)
	}
	return plugin.RunRule(ctx, resolved, verdicts)
}

// gatingAlerts names the alerts that fail the run: error severity only. info and warn are
// reported and pass -- the rule author decides which of its findings is a gate.
func gatingAlerts(alerts []plugin.Alert) []string {
	var out []string
	for _, a := range alerts {
		if a.Severity == plugin.SeverityError {
			out = append(out, a.Plugin+"/"+a.Rule)
		}
	}
	return out
}

// renderAlerts prints the rules section: one line per alert -- severity first, then the
// plugin that raised it, the rule id, and the verdict it read when the rule named one --
// and one line per rule that could not be evaluated, so the report alone explains a
// failed gate even when stderr is captured elsewhere.
func renderAlerts(cmd *cobra.Command, alerts []plugin.Alert, unevaluated []string) error {
	lw := &lineWriter{w: cmd.OutOrStdout()}
	lw.println("rules")
	if len(alerts) == 0 && len(unevaluated) == 0 {
		lw.println("  no alerts")
	}
	for _, a := range alerts {
		lw.printf("  [%-5s] %s/%s: %s%s\n", a.Severity, a.Plugin, a.Rule, a.Message, validatorRef(a.Validator))
	}
	for _, what := range unevaluated {
		lw.printf("  [%-5s] %s: not evaluated -- see the warning above; the gate fails closed\n", "error", what)
	}
	lw.println("")
	return lw.err
}

func validatorRef(validator string) string {
	if validator == "" {
		return ""
	}
	return " (" + validator + ")"
}
