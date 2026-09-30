package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Runner runs gh with args in dir and returns its standard output; Exec is the one that does.
// Tests hand the reader a Runner of their own.
type Runner func(ctx context.Context, dir string, args ...string) ([]byte, error)

const (
	callTimeout = 60 * time.Second
	// maxOutput bounds one gh answer: a full page is well under a megabyte, so anything near this
	// is not an answer to this query.
	maxOutput = 8 << 20
)

// command builds the process; a test points it at a stand-in binary.
var command = func(ctx context.Context, args ...string) *exec.Cmd {
	//nolint:gosec // the program is the constant "gh"; args are assaio's own flags, its query constant and values sent with -f
	return exec.CommandContext(ctx, "gh", args...)
}

// Exec runs the user's own gh, which holds the credentials: assaio passes, reads and stores no
// token. The environment drops GH_REPO and GH_HOST, so the clone -- not a variable left in a
// shell -- decides which repository is read, and turns off prompts, update checks and colour.
func Exec(ctx context.Context, dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	cmd := command(ctx, args...)
	cmd.Dir = dir
	cmd.Env = environment(os.Environ())
	stdout, stderr := &capped{limit: maxOutput}, &capped{limit: 64 << 10}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	switch {
	case errors.Is(err, exec.ErrNotFound):
		return nil, errors.New("evidence --github runs the GitHub CLI, and no gh is on PATH (https://cli.github.com)")
	case stdout.over:
		return nil, fmt.Errorf("gh %s answered with more than %d bytes", args[0], maxOutput)
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return nil, fmt.Errorf("gh %s took longer than %s", args[0], callTimeout)
	case err != nil:
		return stdout.Bytes(), fmt.Errorf("gh %s: %w%s", args[0], err, hint(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func environment(inherited []string) []string {
	out := make([]string, 0, len(inherited)+4)
	for _, kv := range inherited {
		if strings.HasPrefix(kv, "GH_REPO=") || strings.HasPrefix(kv, "GH_HOST=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1", "NO_COLOR=1", "GH_PAGER=cat")
}

// hint is gh's own first line of complaint, with the fix for the one failure everybody meets.
func hint(stderr string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(stderr), "\n")
	if strings.Contains(stderr, "gh auth login") {
		return ": gh is not logged in; run gh auth login"
	}
	if line == "" {
		return ""
	}
	return ": " + line
}

// capped keeps the first limit bytes written to it and remembers whether more came. The buffer is
// a field, not embedded: an embedded bytes.Buffer would lend it ReadFrom, which io.Copy prefers
// over Write, and the cap would never run.
type capped struct {
	buf   bytes.Buffer
	limit int
	over  bool
}

func (c *capped) Write(p []byte) (int, error) {
	if room := c.limit - c.buf.Len(); len(p) > room {
		c.over = true
		if room > 0 {
			c.buf.Write(p[:room])
		}
		return len(p), nil
	}
	return c.buf.Write(p)
}

func (c *capped) Bytes() []byte  { return c.buf.Bytes() }
func (c *capped) String() string { return c.buf.String() }
