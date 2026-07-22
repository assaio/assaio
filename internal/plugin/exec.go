// Package plugin runs out-of-tree parser plugins as subprocesses.
//
// A plugin is any executable that, when invoked as `<command> scan` with
// ASSAIO_PLUGIN_PROTOCOL=1 set, writes to stdout a one-line JSON handshake
// (`{"assaio_plugin":1,"tool":"<name>"}`) followed by zero or more JSONL usage
// records (snake_case, see record.go). The plugin owns its own discovery and parsing;
// assaio owns validation, storage, and pricing. Plugins are opt-in via config only —
// never discovered from PATH.
package plugin

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// killGrace bounds how long a timed-out plugin's I/O may linger after the kill signal:
// exec force-closes the pipes once it elapses, so a grandchild that survived the kill
// cannot stall assaio (see Cmd.WaitDelay).
const killGrace = 2 * time.Second

// Config is one configured plugin: its name, how to invoke it, and its run timeout.
type Config struct {
	Name    string
	Command string
	Timeout time.Duration
}

// Stats summarizes one plugin run: records accepted and lines skipped for failing a
// boundary invariant (see record.go). It does not count the handshake line.
type Stats struct {
	Records int
	Skipped int
}

// Violation is one rejected record line, kept for the plugins verify conformance report.
type Violation struct {
	Line   int
	Reason string
}

// Run invokes cfg's plugin, validates its handshake, and returns the usage records it
// emitted plus Stats. A record line that fails a boundary invariant is skipped and
// counted, not treated as fatal. Handshake failure, timeout, or a non-zero exit is
// returned as an error; the caller (internal/ingest) treats that as the plugin's run
// Failed and continues with the rest.
func Run(ctx context.Context, cfg Config) ([]usage.Record, Stats, error) {
	recs, _, stats, err := run(ctx, cfg, false)
	return recs, stats, err
}

// Verify invokes cfg's plugin like Run, but also collects per-line violations for the
// `plugins verify` conformance report instead of only counting them.
func Verify(ctx context.Context, cfg Config) ([]usage.Record, []Violation, Stats, error) {
	return run(ctx, cfg, true)
}

func run(ctx context.Context, cfg Config, collectViolations bool) ([]usage.Record, []Violation, Stats, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	stdout := &cappedBuffer{cap: maxStdout, cancel: cancel}
	stderr := newPrefixWriter(os.Stderr, "[plugin/"+cfg.Name+"] ")
	//nolint:gosec // the command is the user's own opt-in config entry, resolved via LookPath
	cmd := exec.CommandContext(ctx, cfg.Command, "scan")
	cmd.Env = append(os.Environ(), "ASSAIO_PLUGIN_PROTOCOL=1")
	cmd.WaitDelay = killGrace
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	runErr := cmd.Run()
	stderr.Flush()

	// Check the cap breach before ctx.Err(): an overflow cancels the context itself, so
	// testing ctx.Err() first would misreport a stdout flood as a timeout.
	if stdout.exceeded {
		return nil, nil, Stats{}, fmt.Errorf("plugin %s: stdout exceeded %d bytes", cfg.Name, maxStdout)
	}
	if ctx.Err() != nil {
		return nil, nil, Stats{}, fmt.Errorf("plugin %s: timed out after %s", cfg.Name, cfg.Timeout)
	}
	if runErr != nil {
		return nil, nil, Stats{}, fmt.Errorf("plugin %s: %w", cfg.Name, runErr)
	}

	recs, violations, stats, err := scanOutput(stdout.buf.Bytes(), cfg.Name, collectViolations)
	if err != nil {
		return nil, nil, Stats{}, fmt.Errorf("plugin %s: %w", cfg.Name, err)
	}
	return recs, violations, stats, nil
}

// scanOutput parses a plugin's full stdout: line 1 must be a valid handshake, every
// line after is a candidate usage record. Lines that fail a boundary invariant are
// skipped and counted rather than aborting the scan. collectViolations additionally
// records each skip's line number and reason for the `plugins verify` report.
func scanOutput(out []byte, name string, collectViolations bool) ([]usage.Record, []Violation, Stats, error) {
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), maxStdout)

	if !sc.Scan() {
		return nil, nil, Stats{}, errors.New("no handshake line: plugin produced no output")
	}
	if err := parseHandshake(sc.Bytes(), name); err != nil {
		return nil, nil, Stats{}, err
	}

	var recs []usage.Record
	var violations []Violation
	var stats Stats
	lineNo := 1
	for sc.Scan() {
		lineNo++
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		rec, err := parseRecordLine(line, name)
		if err != nil {
			stats.Skipped++
			if collectViolations {
				violations = append(violations, Violation{Line: lineNo, Reason: err.Error()})
			}
			continue
		}
		recs = append(recs, rec)
		stats.Records++
	}
	if err := sc.Err(); err != nil {
		return nil, nil, Stats{}, fmt.Errorf("reading output: %w", err)
	}
	return recs, violations, stats, nil
}
