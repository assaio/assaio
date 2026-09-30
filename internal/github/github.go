// Package github reads a repository's pull requests through the user's own gh and turns them into
// canonical observations (ADR 0007, ADR 0022): which pull requests exist, their state and merge
// commit, and which commits each lists. It asks one fixed question naming a repository and a
// window, never a commit or a session, and fails rather than return part of an answer.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// maxPages bounds one read; a walk that reaches it says how far back it got.
const maxPages = 20

// Repository is the repository gh resolved for the clone. It is shown to the user, never put in a
// document: its path can hold a person's login (ADR 0020).
type Repository struct {
	Owner, Name, Host string
	IsFork            bool
}

func (r Repository) String() string { return r.Host + "/" + r.Owner + "/" + r.Name }

// Resolve asks gh which repository the clone at root is, the way gh itself decides it.
func Resolve(ctx context.Context, run Runner, root string) (Repository, error) {
	out, err := run(ctx, root, "repo", "view", "--json", "nameWithOwner,url,isFork")
	if err != nil {
		return Repository{}, err
	}
	var view struct {
		NameWithOwner string `json:"nameWithOwner"`
		URL           string `json:"url"`
		IsFork        bool   `json:"isFork"`
	}
	if err := json.Unmarshal(out, &view); err != nil {
		return Repository{}, fmt.Errorf("gh repo view: %w", err)
	}
	owner, name, ok := strings.Cut(view.NameWithOwner, "/")
	u, err := url.Parse(view.URL)
	if !ok || owner == "" || name == "" || err != nil || u.Hostname() == "" {
		return Repository{}, errors.New("gh repo view named no repository")
	}
	return Repository{Owner: owner, Name: name, Host: u.Hostname(), IsFork: view.IsFork}, nil
}

// walk is one read of the repository's pull requests updated at or after since.
type walk struct {
	nodes map[string]node
	// backTo is the update time the read reached: since, unless the page bound stopped it first.
	backTo time.Time
}

// read walks the pull requests newest update first until one was last updated before since, then
// re-reads from the top down to the newest update the walk began with: a pull request updated
// during the walk moved ahead of the cursor and would otherwise be missed. That bound is the
// forge's own clock, so a local clock that runs ahead cannot end the re-read early. The re-read's
// copy of a pull request replaces the first.
func read(ctx context.Context, run Runner, root string, repo Repository, since time.Time) (walk, error) {
	w := walk{nodes: map[string]node{}, backTo: since}
	var newest time.Time
	cursor := ""
	for n := range maxPages {
		p, err := fetch(ctx, run, root, repo, cursor)
		if err != nil {
			return walk{}, err
		}
		if n == 0 && len(p.Nodes) > 0 {
			newest = p.Nodes[0].UpdatedAt
		}
		done := w.keep(p.Nodes, func(nd *node) bool { return nd.UpdatedAt.Before(since) })
		if done || !p.HasNextPage {
			break
		}
		if n == maxPages-1 && len(p.Nodes) > 0 {
			w.backTo = p.Nodes[len(p.Nodes)-1].UpdatedAt
		}
		cursor = p.EndCursor
	}
	if newest.IsZero() {
		return w, nil
	}
	cursor = ""
	for range maxPages {
		p, err := fetch(ctx, run, root, repo, cursor)
		if err != nil {
			return walk{}, err
		}
		if w.keep(p.Nodes, func(nd *node) bool { return !nd.UpdatedAt.After(newest) }) || !p.HasNextPage {
			break
		}
		cursor = p.EndCursor
	}
	return w, nil
}

// keep adds nodes until stop says one is past the walk's end, and reports whether it did.
func (w *walk) keep(nodes []node, stop func(*node) bool) bool {
	for i := range nodes {
		if stop(&nodes[i]) {
			return true
		}
		w.nodes[nodes[i].ID] = nodes[i]
	}
	return false
}

func fetch(ctx context.Context, run Runner, root string, repo Repository, cursor string) (page, error) {
	args := []string{
		"api", "graphql", "--hostname", repo.Host,
		"-f", "query=" + pullRequestQuery, "-f", "owner=" + repo.Owner, "-f", "name=" + repo.Name,
	}
	if cursor != "" {
		args = append(args, "-f", "cursor="+cursor)
	}
	out, runErr := run(ctx, root, args...)
	p, err := parsePage(out)
	var forge *forgeError
	switch {
	case err == nil && runErr == nil:
		return p, nil
	case errors.As(err, &forge), runErr == nil:
		return page{}, fmt.Errorf("reading pull requests of %s: %w", repo, err)
	}
	return page{}, fmt.Errorf("reading pull requests of %s: %w", repo, runErr)
}
