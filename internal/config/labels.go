package config

import (
	"fmt"
	"regexp"
)

// Labels configures the suggested-label derivation: how what a session already recorded
// becomes a proposed annotation. Nothing here ever writes a label on its own.
type Labels struct {
	// Rules are added to the built-in set, which is why a team with its own convention
	// writes one rule rather than restating the common ones. Each entry names a source
	// (branch, skill, agent, entrypoint), an RE2 pattern over that source's value, and the
	// axis value a match implies; the axis vocabularies are closed, so a rule proposing a
	// value outside one fails at load rather than at midnight.
	Rules []LabelRule `koanf:"rules"`
	// Defaults keeps the built-in branch-convention rules. Set it false when a default
	// reads a convention this repository uses for something else -- an `audit/` branch
	// that is not a review, say -- so Rules stands alone.
	Defaults *bool `koanf:"defaults"`
}

// LabelRule mirrors label.Rule as configuration. It is restated here rather than imported
// because this package deliberately depends on nothing inside internal/.
type LabelRule struct {
	Source string `koanf:"source"`
	Match  string `koanf:"match"`
	Axis   string `koanf:"axis"`
	Value  string `koanf:"value"`
}

// KeepDefaults reports whether the built-in rules apply; unset means yes.
func (l Labels) KeepDefaults() bool { return l.Defaults == nil || *l.Defaults }

// Validate checks what this package can check without knowing the label vocabularies: that
// every rule is fully spelled out and its pattern compiles. The axis values are closed sets
// owned by internal/label, and this package deliberately depends on nothing inside
// internal/ -- so `mark --suggest` still rejects a value outside one, and this exists so the
// config surfaces stop reporting a malformed rule set as fine.
func (l Labels) Validate() error {
	for i, r := range l.Rules {
		switch {
		case r.Source == "":
			return fmt.Errorf("rule %d: no source (want branch|skill|agent|entrypoint)", i+1)
		case r.Axis == "":
			return fmt.Errorf("rule %d: no axis", i+1)
		case r.Value == "":
			return fmt.Errorf("rule %d: no value", i+1)
		case r.Match == "":
			return fmt.Errorf("rule %d: no match pattern", i+1)
		}
		if _, err := regexp.Compile(r.Match); err != nil {
			return fmt.Errorf("rule %d: %w", i+1, err)
		}
	}
	return nil
}
