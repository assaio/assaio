package vcs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// maxLineBytes bounds one line of git output. A numstat line is short and a commit header
// carries a subject; anything past this is a repository doing something pathological, and it
// fails the read rather than being buffered without limit.
const maxLineBytes = 1 << 20

// gitOutput runs `git -C root <args>` and returns stdout, wrapping any failure with the
// trimmed stderr so a caller sees why git rejected the command.
func gitOutput(ctx context.Context, root string, args ...string) ([]byte, error) {
	//nolint:gosec // root is the user's own repo path and args are assaio-controlled literals
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return out.Bytes(), nil
}

// RepoRoot resolves repoPath to its git working-tree root, erroring when it is not a repo.
func RepoRoot(ctx context.Context, repoPath string) (string, error) {
	out, err := gitOutput(ctx, repoPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%s is not a git repository: %w", repoPath, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Project is the name observations from root are attributed to -- its basename, matching how
// ingest resolves a session's project from its repository root, so the two line up.
func Project(root string) string { return filepath.Base(root) }

// TouchedFiles lists the paths root's commits changed since `since`. Paths are **local-only**
// by construction: they exist so a caller can read the working tree (blame), and no
// observation ever carries one. Binary edits, which numstat reports as "-", are skipped.
func TouchedFiles(ctx context.Context, root string, since time.Time) ([]string, error) {
	out, err := gitOutput(ctx, root, "log", "--since="+sinceArg(since), "--numstat", "--format=")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var files []string
	sc := newScanner(out)
	for sc.Scan() {
		fields := strings.SplitN(sc.Text(), "\t", 3)
		if len(fields) != 3 || fields[0] == "-" {
			continue
		}
		path := numstatPath(fields[2])
		if _, ok := seen[path]; path == "" || ok {
			continue
		}
		seen[path] = struct{}{}
		files = append(files, path)
	}
	return files, sc.Err()
}

func newScanner(out []byte) *bufio.Scanner {
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	return sc
}

func sinceArg(since time.Time) string { return since.UTC().Format(time.RFC3339) }

// numstatPath resolves a git numstat path field to the file's current path, unwrapping the
// two rename forms git prints ("old => new" and "pre/{old => new}/post") and then git's own
// quoting, so a caller targets the present name on disk rather than the arrow-mangled or
// escaped string. The order matters: git quotes each side of the arrow separately, so the
// arrow has to be resolved before the surviving side is decoded.
func numstatPath(field string) string {
	return unquotePath(renamedTo(field))
}

func renamedTo(field string) string {
	i := strings.Index(field, " => ")
	if i < 0 {
		return field
	}
	if open := strings.LastIndexByte(field[:i], '{'); open >= 0 {
		if rel := strings.IndexByte(field[i:], '}'); rel >= 0 {
			return field[:open] + field[i+len(" => "):i+rel] + field[i+rel+1:]
		}
	}
	return field[i+len(" => "):]
}

// unquotePath decodes the C-style form git wraps a path in whenever core.quotePath is on --
// its default -- so a byte outside ASCII arrives as `"caf\303\251.go"`. Left as-is, that
// string names no file: `git blame` rejects it and path.Ext reads `.go"`, so every non-ASCII
// path silently left the survival rate and landed in the "other" category. The escape
// vocabulary git writes (\\, \", the control letters, and three-digit octal) is a subset of
// Go's, so strconv decodes it; anything it cannot read is returned untouched rather than
// mangled further.
func unquotePath(s string) string {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return s
	}
	if unquoted, err := strconv.Unquote(s); err == nil {
		return unquoted
	}
	return s
}
