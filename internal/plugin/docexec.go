package plugin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// docProtocol is one stdio "document" protocol: a subprocess invoked as `<command>
// <verb>` that reads a single JSON envelope on stdin and writes a one-line handshake
// followed by exactly one JSON document on stdout. The metric (ADR 0004) and rule (ADR
// 0005) protocols differ only in these fields; the parser protocol (ADR 0003) streams
// records instead and keeps its own runner in exec.go.
type docProtocol struct {
	kind      string
	verb      string
	env       string
	maxStdout int
	handshake func(line []byte, wantName string) error
}

// run invokes cfg under p, feeding it stdin, and returns the document that followed a
// valid handshake line. Every protocol failure is an error: the caller drops the
// plugin's output whole rather than acting on something partially sanitized.
func (p docProtocol) run(ctx context.Context, cfg Config, stdin []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	stdout := &cappedBuffer{cap: p.maxStdout, cancel: cancel}
	stderr := newPrefixWriter(os.Stderr, "["+p.kind+"/"+cfg.Name+"] ")
	//nolint:gosec // the command is the user's own opt-in config entry, resolved via LookPath
	cmd := exec.CommandContext(ctx, cfg.Command, p.verb)
	cmd.Env = append(os.Environ(), p.env+"=1")
	cmd.WaitDelay = killGrace
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	runErr := cmd.Run()
	stderr.Flush()

	// Check the cap breach before ctx.Err(): an overflow cancels the context itself, so
	// testing ctx.Err() first would misreport a stdout flood as a timeout.
	if stdout.exceeded {
		return nil, fmt.Errorf("%s plugin %s: stdout exceeded %d bytes", p.kind, cfg.Name, p.maxStdout)
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("%s plugin %s: timed out after %s", p.kind, cfg.Name, cfg.Timeout)
	}
	if runErr != nil {
		return nil, fmt.Errorf("%s plugin %s: %w", p.kind, cfg.Name, runErr)
	}

	line, doc := splitHandshakeLine(stdout.buf.Bytes())
	if len(bytes.TrimSpace(line)) == 0 {
		return nil, fmt.Errorf("%s plugin %s: no handshake line: plugin produced no output", p.kind, cfg.Name)
	}
	if err := p.handshake(line, cfg.Name); err != nil {
		return nil, fmt.Errorf("%s plugin %s: %w", p.kind, cfg.Name, err)
	}
	return doc, nil
}

// splitHandshakeLine returns stdout's first line and everything after it; a plugin that
// wrote only a handshake yields an empty rest, which the document parser reports as a
// missing document.
func splitHandshakeLine(out []byte) (line, rest []byte) {
	if idx := bytes.IndexByte(out, '\n'); idx >= 0 {
		return out[:idx], out[idx+1:]
	}
	return out, nil
}

// violationSuffix folds the first violation into the error text so a bare run failure
// names the offending check without needing a verify report.
func violationSuffix(violations []string) string {
	if len(violations) == 0 {
		return ""
	}
	return ": " + violations[0]
}
