package vcs

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// lazyFetchUnsafe reports a partial clone read by a git older than 2.44, which ignores
// GIT_NO_LAZY_FETCH: reading a commit it lacks would fetch it with the user's credentials.
func lazyFetchUnsafe(ctx context.Context, root string) (bool, error) {
	_, err := gitOutput(ctx, root, "config", "--get-regexp", `^(extensions\.partialclone|remote\..*\.promisor)$`)
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return false, nil
	case err != nil:
		return false, err
	}
	out, err := gitOutput(ctx, root, "version")
	if err != nil {
		return false, err
	}
	return !honoursNoLazyFetch(string(out)), nil
}

// honoursNoLazyFetch reads `git version` output; a version it cannot read is treated as too old.
func honoursNoLazyFetch(version string) bool {
	fields := strings.Fields(version)
	if len(fields) < 3 {
		return false
	}
	parts := strings.SplitN(fields[2], ".", 3)
	if len(parts) < 2 {
		return false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	return major > 2 || major == 2 && minor >= 44
}
