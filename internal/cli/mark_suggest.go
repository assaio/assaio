package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/label"
	"github.com/assaio/assaio/internal/store"
)

// suggestion pairs a session with the axes derived for it, carrying the evidence so the
// render can show why rather than only what.
type suggestion struct {
	ref  store.SessionRef
	tool string
	axes []label.Suggestion
}

// newLabelEngine builds the derivation from the built-in rules plus whatever the config
// adds. Configured rules come last, so a repository's own convention is read after the
// common ones -- and disagreeing with a default yields nothing for that axis rather than
// letting either win silently.
func newLabelEngine(cfg *config.Config) (*label.Engine, error) {
	configured := make([]label.Rule, 0, len(cfg.Labels.Rules))
	for _, r := range cfg.Labels.Rules {
		configured = append(configured, label.Rule{Source: r.Source, Match: r.Match, Axis: r.Axis, Value: r.Value})
	}
	// Validated alone first, so a broken rule is numbered as the person wrote it rather
	// than by its position after the built-in ones.
	if _, err := label.NewEngine(configured); err != nil {
		return nil, fmt.Errorf("labels.rules: %w", err)
	}
	if !cfg.Labels.KeepDefaults() {
		return label.NewEngine(configured)
	}
	rules := make([]label.Rule, 0, len(label.DefaultRules)+len(configured))
	rules = append(rules, label.DefaultRules...)
	rules = append(rules, configured...)
	return label.NewEngine(rules)
}

func runMarkSuggest(cmd *cobra.Command, st *store.Store, since string, accept bool) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	engine, err := newLabelEngine(&cfg)
	if err != nil {
		return err
	}
	start, err := parseSinceAt(since, time.Now())
	if err != nil {
		return err
	}
	sessions, err := st.Sessions(cmd.Context(), start)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		cmd.Println(emptyStoreHint(cmd, "No sessions in this window."))
		return nil
	}
	signals, err := st.SessionSignals(cmd.Context(), start)
	if err != nil {
		return err
	}
	found := deriveSuggestions(engine, sessions, signals)
	if !accept {
		return renderSuggestions(cmd, found, sessions, since)
	}
	return applySuggestions(cmd, st, found)
}

// deriveSuggestions runs the engine over each session, keeping only axes the person has not
// already answered: a hand-made label is the authority, and a derived one never overwrites
// or merges into it.
func deriveSuggestions(engine *label.Engine, sessions []store.SessionRow, signals []store.SessionSignalRow) []suggestion {
	bySession := collectSignals(signals)
	out := make([]suggestion, 0, len(sessions))
	for i := range sessions {
		s := &sessions[i]
		fresh := unlabeledAxes(engine.Derive(bySession[signalKey{s.SessionID, s.Member}]), s)
		if len(fresh) == 0 {
			continue
		}
		out = append(out, suggestion{
			ref:  store.SessionRef{SessionID: s.SessionID, Member: s.Member, Project: s.Project, LastTs: s.LastTs},
			tool: s.Tool, axes: fresh,
		})
	}
	return out
}

// signalKey pairs a session with the member who produced it, which is what makes a session
// unique on a store holding more than one machine's rows.
type signalKey struct{ session, member string }

func collectSignals(rows []store.SessionSignalRow) map[signalKey]label.Signals {
	out := make(map[signalKey]label.Signals)
	for i := range rows {
		r := &rows[i]
		k := signalKey{r.SessionID, r.Member}
		sig, ok := out[k]
		if !ok {
			sig = label.Signals{}
			out[k] = sig
		}
		addSignal(sig, label.SourceBranch, r.Branch)
		addSignal(sig, label.SourceSkill, r.Skill)
		addSignal(sig, label.SourceAgent, r.Agent)
		addSignal(sig, label.SourceEntrypoint, r.Entrypoint)
	}
	return out
}

// addSignal keeps each source's values distinct, so one branch recorded on four hundred
// rows is one value and two different branches stay two.
func addSignal(sig label.Signals, source, value string) {
	if value == "" {
		return
	}
	for _, seen := range sig[source] {
		if seen == value {
			return
		}
	}
	sig[source] = append(sig[source], value)
}

// unlabeledAxes keeps suggestions only for a session nobody has annotated at all. An empty
// axis on a session that carries any label is not an unanswered question: clearing one axis
// leaves the row in place, so "no task here" is a decision, and filling it back in would
// overwrite the person's answer with a derived one.
func unlabeledAxes(derived []label.Suggestion, s *store.SessionRow) []label.Suggestion {
	if s.Task != "" || s.Outcome != "" || s.Difficulty != "" {
		return nil
	}
	return derived
}

// applySuggestions writes the derived axes, merging over whatever each session already
// carries so an outcome set by hand survives a task derived later.
func applySuggestions(cmd *cobra.Command, st *store.Store, found []suggestion) error {
	if len(found) == 0 {
		cmd.Println("Nothing to accept: no session in this window derives a label it does not already carry.")
		return nil
	}
	for _, f := range found {
		current, _, err := st.Label(cmd.Context(), f.ref)
		if err != nil {
			return err
		}
		current.MarkedAt = time.Now()
		for _, a := range f.axes {
			switch a.Axis {
			case label.Task:
				current.Task = a.Value
			case label.Outcome:
				current.Outcome = a.Value
			case label.Difficulty:
				current.Difficulty = a.Value
			}
		}
		if err := st.Mark(cmd.Context(), f.ref, current); err != nil {
			return err
		}
	}
	cmd.Printf("accepted %d suggestion(s)\n", len(found))
	cmd.Println("Review them with 'mark --list'; change any with 'mark <id> --task …'.")
	return nil
}
