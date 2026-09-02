package plugin

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/config"
)

// Resolve validates a config.PluginConfig and turns it into a Config ready for Run:
// the command is resolved to an absolute path via exec.LookPath when it is not already
// one. Validation happens here so every caller path (ingest, verify) enforces it.
func Resolve(pc config.PluginConfig) (Config, error) { return resolve(pc, pc.Validate) }

// ResolveMetric is Resolve for an entry under `metrics:`, the one kind that may declare the
// inputs it reads.
func ResolveMetric(pc config.PluginConfig) (Config, error) { return resolve(pc, pc.ValidateMetric) }

func resolve(pc config.PluginConfig, validate func() error) (Config, error) {
	if err := validate(); err != nil {
		return Config{}, fmt.Errorf("plugin %q: %w", pc.Name, err)
	}
	timeout, err := pc.TimeoutOrDefault()
	if err != nil {
		return Config{}, fmt.Errorf("plugin %s: invalid timeout %q: %w", pc.Name, pc.Timeout, err)
	}
	// Every check on what the config *says* runs before the one that touches the filesystem:
	// a typo'd capability is the same mistake whether or not the command happens to exist, and
	// reporting the missing binary first hides it behind an error about a different line.
	allow, err := capabilities(pc)
	if err != nil {
		return Config{}, fmt.Errorf("plugin %s: %w", pc.Name, err)
	}

	command := pc.Command
	if !filepath.IsAbs(command) {
		resolved, err := exec.LookPath(command)
		if err != nil {
			return Config{}, fmt.Errorf("plugin %s: command %q not found: %w", pc.Name, command, err)
		}
		command = resolved
	}
	return Config{Name: pc.Name, Command: command, Timeout: timeout, Allow: allow}, nil
}

// capabilities maps a config entry's `needs:` list onto the closed vocabulary, rejecting a
// name this build does not know. The check lives here rather than in internal/config because
// the vocabulary is internal/analyze's, and every package that reads a config would otherwise
// drag the validator registry in behind it.
func capabilities(pc config.PluginConfig) ([]analyze.Capability, error) {
	out := make([]analyze.Capability, len(pc.Needs))
	for i, n := range pc.Needs {
		c := analyze.Capability(n)
		if !analyze.ValidCapability(c) {
			return nil, fmt.Errorf("unknown need %q (want one of %s)", n, strings.Join(capabilityNames(), ", "))
		}
		out[i] = c
	}
	return out, nil
}

// capabilityNames renders the closed set for an error message, read from the registry rather
// than repeated here.
func capabilityNames() []string {
	caps := analyze.Capabilities()
	names := make([]string, len(caps))
	for i, c := range caps {
		names[i] = string(c)
	}
	return names
}
