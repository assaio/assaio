package attribution

import (
	"context"
	"fmt"
	"time"
)

// Identities a fixture commits as: the corpus's own, and the one GitHub.com records on the
// commits it writes. A squash is authored under the pull-request author's noreply address, which
// the collector must not mistake for the forge.
const (
	corpusIdentity  = "Corpus <corpus@assaio.test>"
	forgeName       = "GitHub"
	forgeEmail      = "noreply@github.com"
	prAuthorName    = "Corpus"
	prAuthorNoreply = "12345+corpus@users.noreply.github.com"
)

// identity pins who wrote and who committed a spec, and when, through the environment -- the one
// place every git version reads both from.
func identity(spec *commitSpec) []string {
	authorName, authorEmail := splitIdentity(corpusIdentity)
	if spec.Author != "" {
		authorName, authorEmail = splitIdentity(spec.Author)
	}
	if spec.Lands == "squash" && spec.Forge {
		authorName, authorEmail = prAuthorName, prAuthorNoreply
	}
	committerName, committerEmail := splitIdentity(corpusIdentity)
	if spec.Forge {
		committerName, committerEmail = forgeName, forgeEmail
	}
	authored := spec.At
	if spec.Authored != nil {
		authored = *spec.Authored
	}
	return []string{
		"GIT_AUTHOR_NAME=" + authorName, "GIT_AUTHOR_EMAIL=" + authorEmail,
		"GIT_AUTHOR_DATE=" + stamp(authored),
		"GIT_COMMITTER_NAME=" + committerName, "GIT_COMMITTER_EMAIL=" + committerEmail,
		"GIT_COMMITTER_DATE=" + stamp(spec.At),
	}
}

// committerOnly keeps the committer half of an identity: a replayed commit keeps the author it
// was written by, which is the property a rebase-merge fixture tests.
func committerOnly(env []string) []string {
	out := make([]string, 0, len(env))
	for _, v := range env {
		if len(v) > len("GIT_COMMITTER_") && v[:len("GIT_COMMITTER_")] == "GIT_COMMITTER_" {
			out = append(out, v)
		}
	}
	return out
}

func stamp(offset time.Duration) string { return epoch.Add(offset).Format(time.RFC3339) }

func splitIdentity(id string) (name, email string) {
	for i := range id {
		if id[i] == '<' {
			return id[:i-1], id[i+1 : len(id)-1]
		}
	}
	return id, ""
}

// land brings work onto the main line as a forge merges it. Squash and merge are written with
// plumbing (commit-tree, update-ref) so no merge strategy, fast-forward setting or default
// message of the installed git decides the fixture; a rebase replays one commit with
// cherry-pick, which keeps its author and takes the committer from the environment.
func land(ctx context.Context, dir string, spec *commitSpec, hashes map[string]string) (string, error) {
	if err := git(ctx, dir, "checkout", "-f", mainBranch); err != nil {
		return "", err
	}
	if spec.Lands == "rebase" {
		source := hashes[spec.From]
		if source == "" {
			return "", fmt.Errorf("rebase of unknown commit %q", spec.From)
		}
		if err := gitEnv(ctx, dir, committerOnly(identity(spec)), "cherry-pick", source); err != nil {
			return "", err
		}
		return revParse(ctx, dir, "HEAD")
	}
	parents := []string{"-p", mainBranch}
	switch spec.Lands {
	case "squash":
	case "merge":
		parents = append(parents, "-p", spec.From)
	default:
		return "", fmt.Errorf("unknown landing %q", spec.Lands)
	}
	args := append([]string{"commit-tree", spec.From + "^{tree}"}, parents...)
	hash, err := gitOut(ctx, dir, identity(spec), append(args, "-m", spec.Tag)...)
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}
	if err := git(ctx, dir, "update-ref", "refs/heads/"+mainBranch, hash); err != nil {
		return "", err
	}
	return hash, git(ctx, dir, "checkout", "-f", mainBranch)
}
