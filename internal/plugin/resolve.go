package plugin

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/assaio/assaio/internal/config"
)

// Resolve validates a config.PluginConfig and turns it into a Config ready for Run:
// the command is resolved to an absolute path via exec.LookPath when it is not already
// one. Validation happens here so every caller path (ingest, verify) enforces it.
func Resolve(pc config.PluginConfig) (Config, error) {
	if err := pc.Validate(); err != nil {
		return Config{}, fmt.Errorf("plugin %q: %w", pc.Name, err)
	}
	timeout, err := pc.TimeoutOrDefault()
	if err != nil {
		return Config{}, fmt.Errorf("plugin %s: invalid timeout %q: %w", pc.Name, pc.Timeout, err)
	}

	command := pc.Command
	if !filepath.IsAbs(command) {
		resolved, err := exec.LookPath(command)
		if err != nil {
			return Config{}, fmt.Errorf("plugin %s: command %q not found: %w", pc.Name, command, err)
		}
		command = resolved
	}

	return Config{Name: pc.Name, Command: command, Timeout: timeout}, nil
}
