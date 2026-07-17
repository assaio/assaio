package config

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// pluginNamePattern constrains a plugin's config name, which also becomes its namespace
// (tool plugin:<name> for parser plugins, validator plugin:<name> for metric plugins).
var pluginNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// defaultPluginTimeout applies when a configured plugin omits timeout.
const defaultPluginTimeout = 60 * time.Second

// PluginConfig is one exec plugin as declared in config.yaml -- the same entry shape
// serves both the plugins: (parser) and metrics: (analyzer) lists.
type PluginConfig struct {
	// Name becomes the plugin:<name> namespace on whatever the plugin emits.
	Name string `koanf:"name"`
	// Command is the plugin executable; resolved via exec.LookPath if not absolute.
	Command string `koanf:"command"`
	// Timeout bounds one plugin invocation, e.g. "60s". Defaults to 60s if empty.
	Timeout string `koanf:"timeout"`
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
	return nil
}

// TimeoutOrDefault parses Timeout, defaulting to 60s when empty.
func (p PluginConfig) TimeoutOrDefault() (time.Duration, error) {
	if p.Timeout == "" {
		return defaultPluginTimeout, nil
	}
	return time.ParseDuration(p.Timeout)
}

// dupName returns the first name declared twice within one plugin list. plugins: and
// metrics: are separate namespaces on purpose -- one binary may serve both protocols
// under one name.
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
