package vcs

import (
	"context"
	"strconv"
	"strings"
)

// FirstParents returns up to n commits along the first-parent line from hash, hash first: the
// order in which a forge's rebase merge lays a pull request's commits onto the branch.
func FirstParents(ctx context.Context, root, hash string, n int) ([]string, error) {
	out, err := gitOutput(ctx, root, "rev-list", "--first-parent", "--max-count="+strconv.Itoa(n), hash)
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}
