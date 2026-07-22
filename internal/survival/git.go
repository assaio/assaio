package survival

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

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

// repoRoot resolves repoPath to its git working-tree root, erroring when it is not a repo.
func repoRoot(ctx context.Context, repoPath string) (string, error) {
	out, err := gitOutput(ctx, repoPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%s is not a git repository: %w", repoPath, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// windowCommits is the set of commit hashes reachable from HEAD with a commit date at or
// after since -- the commits whose surviving lines we count.
func windowCommits(ctx context.Context, root string, since time.Time) (map[string]struct{}, error) {
	out, err := gitOutput(ctx, root, "log", "--no-merges", "--since="+since.Format(time.RFC3339), "--format=%H")
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		if h := strings.TrimSpace(sc.Text()); h != "" {
			set[h] = struct{}{}
		}
	}
	return set, sc.Err()
}

// addedAndTouched sums the lines the window's commits added (git numstat) and returns the
// set of files they touched, for the survival blame pass. Binary edits (numstat "-") are
// skipped.
func addedAndTouched(ctx context.Context, root string, since time.Time) (added int64, files []string, err error) {
	out, err := gitOutput(ctx, root, "log", "--since="+since.Format(time.RFC3339), "--numstat", "--format=")
	if err != nil {
		return 0, nil, err
	}
	seen := make(map[string]struct{})
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		fields := strings.SplitN(sc.Text(), "\t", 3)
		if len(fields) != 3 || fields[0] == "-" { // "-" marks a binary edit: no line counts
			continue
		}
		if n, e := strconv.ParseInt(fields[0], 10, 64); e == nil {
			added += n
		}
		if path := numstatPath(fields[2]); path != "" {
			if _, ok := seen[path]; !ok {
				seen[path] = struct{}{}
				files = append(files, path)
			}
		}
	}
	return added, files, sc.Err()
}

// survivingLines blames each still-present touched file at HEAD and counts lines whose
// commit is in the window set -- window-authored lines still in the tree. A file git can't
// blame (deleted or renamed away) is skipped; its lines legitimately did not survive.
func survivingLines(ctx context.Context, root string, files []string, commits map[string]struct{}) (surviving int64, blamed int, err error) {
	for _, f := range files {
		out, e := gitOutput(ctx, root, "blame", "--line-porcelain", "HEAD", "--", f)
		if e != nil {
			if ctx.Err() != nil {
				return surviving, blamed, ctx.Err()
			}
			continue // a deleted/renamed-away/binary path can't be blamed: it didn't survive
		}
		blamed++
		sc := bufio.NewScanner(bytes.NewReader(out))
		sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
		for sc.Scan() {
			// --line-porcelain emits one "<sha> <orig> <final> ..." header per file line;
			// content lines start with a tab and metadata with a keyword, so only a header's
			// first token is ever a window hash. Matching the whole token fits SHA-1 and -256.
			line := sc.Text()
			if sp := strings.IndexByte(line, ' '); sp > 0 {
				if _, ok := commits[line[:sp]]; ok {
					surviving++
				}
			}
		}
		if sc.Err() != nil {
			continue // a pathological (e.g. >8 MB) line breaks this file's scan, not the report
		}
	}
	return surviving, blamed, nil
}

// numstatPath resolves a git numstat path field to the file's current path, unwrapping the
// two rename forms git prints ("old => new" and "pre/{old => new}/post") so the blame pass
// targets the present name, not the arrow-mangled string.
func numstatPath(field string) string {
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
