package vcs

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

// TestCollectReadsAuthorTimeAndTheForgeCommitter: the author time survives a rewrite, and only the
// forge's own committer identity is recognised -- not a user's personal noreply address, and not
// an identity a repository's .mailmap rewrote in either direction.
func TestCollectReadsAuthorTimeAndTheForgeCommitter(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "--initial-branch=main")
	write(t, dir, ".mailmap", "Maintainer <maintainer@example.test> <noreply@github.com>\n"+
		"GitHub <noreply@github.com> <dev@example.test>\n")
	base := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name             string
		authored, commit time.Time
		committer        string
		wantForge        bool
	}{
		{"written and committed at once", base, base, "Dev <dev@example.test>", false},
		{"amended three days later", base.Add(time.Hour), base.Add(72 * time.Hour), "Dev <dev@example.test>", false},
		{"squashed by the forge", base.Add(80 * time.Hour), base.Add(80 * time.Hour), "GitHub <noreply@github.com>", true},
		{"the forge address in another case", base.Add(81 * time.Hour), base.Add(81 * time.Hour), "GITHUB <NoReply@GitHub.com>", true},
		{"a user's own noreply address", base.Add(82 * time.Hour), base.Add(82 * time.Hour), "Dev <123+dev@users.noreply.github.com>", false},
	}
	for _, tt := range tests {
		write(t, dir, tt.name+".txt", "x\n")
		runGit(t, dir, "add", ".")
		name, email := splitIdentity(tt.committer)
		commitAs(t, dir, tt.authored, tt.commit, name, email, tt.name)
	}
	events, skipped, err := Collect(context.Background(), dir, "repo", allTime, observedAt, "test")
	if err != nil || skipped != 0 {
		t.Fatalf("Collect: %v, skipped %d", err, skipped)
	}
	if len(events) != len(tests) {
		t.Fatalf("%d observations, want %d", len(events), len(tests))
	}
	byTime := map[time.Time]event.Commit{}
	for i := range events {
		byTime[events[i].OccurredAt] = events[i].Payload.(event.Commit)
	}
	for _, tt := range tests {
		c, ok := byTime[tt.commit.UTC()]
		if !ok {
			t.Fatalf("%s: no observation at its committer time %s", tt.name, tt.commit)
		}
		if !c.AuthoredAt.Equal(tt.authored) || c.CommittedByForge != tt.wantForge {
			t.Errorf("%s: authored %s forge %v, want %s %v", tt.name, c.AuthoredAt, c.CommittedByForge, tt.authored, tt.wantForge)
		}
	}
}

func splitIdentity(identity string) (name, email string) {
	for i := range identity {
		if identity[i] == '<' {
			return identity[:i-1], identity[i+1 : len(identity)-1]
		}
	}
	return identity, ""
}

func commitAs(t *testing.T, dir string, authored, committed time.Time, committerName, committerEmail, message string) {
	t.Helper()
	//nolint:gosec // test-only git driver over a t.TempDir() path, not user input
	cmd := exec.CommandContext(context.Background(), "git", "-C", dir, "commit", "-m", message)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=Dev", "GIT_AUTHOR_EMAIL=dev@example.test",
		"GIT_AUTHOR_DATE="+authored.Format(time.RFC3339),
		"GIT_COMMITTER_NAME="+committerName, "GIT_COMMITTER_EMAIL="+committerEmail,
		"GIT_COMMITTER_DATE="+committed.Format(time.RFC3339))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}
