// Package config loads assaio's runtime configuration from defaults, an
// optional YAML file, and ASSAIO_-prefixed environment variables.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

// sincePattern matches the day-window form accepted by report/clear, e.g. "7d" or "0d".
var sincePattern = regexp.MustCompile(`^\d+d$`)

// Config holds the merged CLI/report configuration.
type Config struct {
	// Since is the report lookback window, e.g. "30d".
	Since string `koanf:"since"`
	// Format is the report output format, e.g. "table", "json", "csv".
	Format string `koanf:"format"`
	// Sources overrides the built-in default log-file locations per tool. See the
	// Sources type doc for the resolution rule.
	Sources Sources `koanf:"sources"`
	// Plugins lists out-of-tree exec parser plugins, opt-in only: never discovered from
	// PATH.
	Plugins []PluginConfig `koanf:"plugins"`
	// Metrics lists out-of-tree exec metric plugins (custom analyzers), opt-in only,
	// same entry shape as Plugins. Run by analyze and dashboard; the team server never
	// executes them (see internal/plugin's metric protocol and ADR 0004).
	Metrics []PluginConfig `koanf:"metrics"`
	// Privacy governs shareable exports; only the dashboard command reads it today.
	Privacy Privacy `koanf:"privacy"`
	// Server holds `assaio-agent serve` defaults.
	Server Server `koanf:"server"`
	// Sync holds `assaio-agent sync` defaults.
	Sync Sync `koanf:"sync"`
	// Pricing lets a subscription or negotiated-rate user declare their real cost basis;
	// unset keeps every cost an API pay-as-you-go estimate. See the Pricing type.
	Pricing Pricing `koanf:"pricing"`
}

// Sources overrides the filesystem roots assaio discovers each tool's session logs
// under (internal/paths' built-in defaults). For a given tool, a non-empty list
// replaces the default roots entirely — it is never merged with them, so the result is
// always exactly what the user configured. An empty list (the default) keeps the
// current, built-in locations.
type Sources struct {
	Claude []string `koanf:"claude"`
	Codex  []string `koanf:"codex"`
	Gemini []string `koanf:"gemini"`
	Cline  []string `koanf:"cline"`
}

// Privacy holds settings for shareable exports, distinct from the interactive CLI
// tables (status/report/effectiveness), which always show real names.
type Privacy struct {
	// Anonymize pseudonymizes project names, and member names in the dashboard's Team
	// section, in the dashboard HTML (including the team server's served dashboard),
	// since those are meant to be shared; it does not affect any other command.
	// Defaults to true.
	Anonymize bool `koanf:"anonymize"`
}

// Server holds `assaio-agent serve` defaults; explicit flags still take precedence. The
// team server is the one non-offline exception in assaio: a self-hosted process you run
// and control yourself (see internal/server's package doc for the MVP security
// boundary), not a hosted assaio service.
type Server struct {
	// Addr is the listen address, e.g. ":8787". Override with ASSAIO_SERVER_ADDR.
	Addr string `koanf:"addr"`
	// Token is the shared bearer secret clients must present. Override with
	// ASSAIO_SERVER_TOKEN rather than committing a real token to a config file.
	Token string `koanf:"token"`
}

// Sync holds `assaio-agent sync` defaults; explicit flags still take precedence.
type Sync struct {
	// Server is the team server's base URL, e.g. "http://localhost:8787".
	Server string `koanf:"server"`
	// Token is the shared bearer secret to present to Server. Override with
	// ASSAIO_SYNC_TOKEN.
	Token string `koanf:"token"`
	// Member self-identifies this machine to the server. Empty (the default) makes
	// sync derive a stable pseudonym from hostname+OS-user instead -- pseudonymized is
	// assaio's default privacy mode; setting Member is a deliberate opt-in.
	Member string `koanf:"member"`
}

func defaults() Config {
	return Config{Since: "30d", Format: "table", Privacy: Privacy{Anonymize: true}}
}

// Load merges defaults < config file at path (if present) < ASSAIO_-prefixed
// env vars, in that order, and returns the resulting Config.
func Load(path string) (Config, error) {
	k := koanf.New(".")
	if err := k.Load(structs.Provider(defaults(), "koanf"), nil); err != nil {
		return Config{}, err
	}
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return Config{}, err
		}
	}
	envProvider := env.Provider("ASSAIO_", ".", envKeyResolver())
	if err := k.Load(envProvider, nil); err != nil {
		return Config{}, err
	}
	var c Config
	if err := k.Unmarshal("", &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Validate reports whether Format, Since, Plugins, Metrics, and Pricing hold values the
// CLI accepts.
//
//nolint:gocritic // Config is a small value bundle validated once per CLI run, not a hot path.
func (c Config) Validate() error {
	switch c.Format {
	case "table", "json", "csv":
	default:
		return fmt.Errorf("invalid format %q (want table|json|csv)", c.Format)
	}
	if !sincePattern.MatchString(c.Since) {
		return fmt.Errorf("invalid since %q (want e.g. 7d)", c.Since)
	}
	for _, p := range c.Plugins {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("plugin %q: %w", p.Name, err)
		}
	}
	for _, m := range c.Metrics {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("metric %q: %w", m.Name, err)
		}
	}
	if name := dupName(c.Plugins); name != "" {
		return fmt.Errorf("duplicate plugin name %q", name)
	}
	if name := dupName(c.Metrics); name != "" {
		return fmt.Errorf("duplicate metric name %q", name)
	}
	if err := c.Pricing.Validate(); err != nil {
		return err
	}
	return nil
}
