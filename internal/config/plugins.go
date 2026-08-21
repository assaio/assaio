package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/analyze"
)

// pluginNamePattern constrains a plugin's config name, which also becomes its namespace
// (tool plugin:<name> for parser plugins, validator plugin:<name> for metric plugins,
// the alert's raising plugin for rule plugins).
var pluginNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// defaultPluginTimeout applies when a configured plugin omits timeout.
const defaultPluginTimeout = 60 * time.Second

// PluginConfig is one exec plugin as declared in config.yaml -- the same entry shape
// serves the plugins: (parser), metrics: (analyzer), and rules: (gate) lists.
type PluginConfig struct {
	// Name becomes the plugin:<name> namespace on whatever the plugin emits.
	Name string `koanf:"name"`
	// Command is the plugin executable; resolved via exec.LookPath if not absolute.
	Command string `koanf:"command"`
	// Timeout bounds one plugin invocation, e.g. "60s". Defaults to 60s if empty.
	Timeout string `koanf:"timeout"`
	// Needs is what a metric plugin declares it reads, from assaio's closed capability set.
	// Everything in the envelope is sent unasked except the step timeline, which was measured
	// at ~44 MB on a real store: a plugin that wants it lists "trace" here, and one that does
	// not is told the section was withheld rather than handed an empty array it could mistake
	// for a window with no sequences (B168). Ignored for parser and rule plugins, which read a
	// different document.
	Needs []string `koanf:"needs"`
}

// Validate checks one plugin's name, command, and timeout.
func (p PluginConfig) Validate() error {
	if !pluginNamePattern.MatchString(p.Name) {
		return fmt.Errorf("invalid name %q (want [a-z0-9-]+)", p.Name)
	}
	if p.Command == "" {
		return errors.New("command is required")
	}
	if _, err := p.TimeoutOrDefault(); err != nil {
		return fmt.Errorf("invalid timeout %q: %w", p.Timeout, err)
	}
	if len(p.Needs) > 0 {
		return errors.New("needs: applies to metric plugins only; a parser or rule plugin reads a different document")
	}
	return nil
}

// ValidateMetric is Validate for an entry under `metrics:`, which is the one kind that may
// declare what it reads. Kept as a second method rather than a flag on the first so the
// generic path cannot accidentally accept a key it would then ignore -- silently ignored
// configuration is how a user ends up sure they asked for something they never got.
func (p PluginConfig) ValidateMetric() error {
	needs := p.Needs
	p.Needs = nil
	if err := p.Validate(); err != nil {
		return err
	}
	for _, need := range needs {
		if !analyze.ValidCapability(analyze.Capability(need)) {
			return fmt.Errorf("unknown need %q (want one of %s)", need, strings.Join(capabilityNames(), ", "))
		}
	}
	return nil
}

// capabilityNames renders the closed capability set for an error message, read from the
// registry rather than repeated here.
func capabilityNames() []string {
	caps := analyze.Capabilities()
	names := make([]string, len(caps))
	for i, c := range caps {
		names[i] = string(c)
	}
	return names
}

// TimeoutOrDefault parses Timeout, defaulting to 60s when empty.
func (p PluginConfig) TimeoutOrDefault() (time.Duration, error) {
	if p.Timeout == "" {
		return defaultPluginTimeout, nil
	}
	return time.ParseDuration(p.Timeout)
}

// dupName returns the first name declared twice within one plugin list. plugins:,
// metrics:, and rules: are separate namespaces on purpose -- one binary may serve
// several protocols under one name.
func dupName(list []PluginConfig) string {
	seen := make(map[string]bool, len(list))
	for _, p := range list {
		if seen[p.Name] {
			return p.Name
		}
		seen[p.Name] = true
	}
	return ""
}
