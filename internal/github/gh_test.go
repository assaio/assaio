package github

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelperProcess is the stand-in gh the tests below run: it is this test binary, re-entered.
func TestHelperProcess(t *testing.T) {
	mode := os.Getenv("ASSAIO_GH_HELPER")
	if mode == "" {
		return
	}
	switch mode {
	case "env":
		wd, _ := os.Getwd()
		fmt.Printf("%s|%s|%s", os.Getenv("GH_REPO"), os.Getenv("GH_PROMPT_DISABLED"), filepath.Base(wd))
	case "auth":
		fmt.Fprintln(os.Stderr, "To get started with GitHub CLI, please run:  gh auth login")
		os.Exit(4)
	}
	os.Exit(0)
}

func helper(t *testing.T, mode string) {
	t.Helper()
	t.Setenv("ASSAIO_GH_HELPER", mode)
	previous := command
	command = func(ctx context.Context, args ...string) *exec.Cmd {
		//nolint:gosec // re-runs this test binary as a stand-in gh
		return exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestHelperProcess$", "--"}, args...)...)
	}
	t.Cleanup(func() { command = previous })
}

// TestGhRunsInTheCloneWithoutTheShellsRepository: GH_REPO would make gh read another repository's
// pull requests and report this clone's as never listed.
func TestGhRunsInTheCloneWithoutTheShellsRepository(t *testing.T) {
	helper(t, "env")
	t.Setenv("GH_REPO", "someone/else")
	dir := filepath.Join(t.TempDir(), "clone")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	out, err := Exec(context.Background(), dir, "repo", "view")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out); got != "|1|clone" {
		t.Fatalf("gh saw GH_REPO|GH_PROMPT_DISABLED|dir = %q, want no GH_REPO, prompts off, run in the clone", got)
	}
}

func TestAnUnauthenticatedGhSaysHowToFixIt(t *testing.T) {
	helper(t, "auth")
	_, err := Exec(context.Background(), t.TempDir(), "api", "graphql")
	if err == nil || !strings.Contains(err.Error(), "run gh auth login") {
		t.Fatalf("err = %v, want the login fix", err)
	}
}

func TestAMissingGhSaysWhatIsMissing(t *testing.T) {
	previous := command
	command = func(ctx context.Context, args ...string) *exec.Cmd {
		//nolint:gosec // a constant program name that is meant not to exist
		return exec.CommandContext(ctx, "assaio-no-such-gh", args...)
	}
	t.Cleanup(func() { command = previous })
	_, err := Exec(context.Background(), t.TempDir(), "repo", "view")
	if err == nil || !strings.Contains(err.Error(), "no gh is on PATH") {
		t.Fatalf("err = %v, want the missing client named", err)
	}
}

// TestOutputPastTheCapIsRefusedNotTruncated copies the way exec does, through io.Copy, which
// takes any faster path the writer offers.
func TestOutputPastTheCapIsRefusedNotTruncated(t *testing.T) {
	c := &capped{limit: 4}
	if n, err := io.Copy(c, strings.NewReader("abcdef")); n != 6 || err != nil || !c.over || c.String() != "abcd" {
		t.Fatalf("copied %d, %v; over %v, kept %q", n, err, c.over, c.String())
	}
}

func TestTheEnvironmentDropsRepositoryAndHost(t *testing.T) {
	env := strings.Join(environment([]string{"GH_REPO=a/b", "GH_HOST=x", "GH_TOKEN=kept", "PATH=/bin"}), " ")
	if strings.Contains(env, "GH_REPO") || strings.Contains(env, "GH_HOST") || !strings.Contains(env, "GH_TOKEN=kept") {
		t.Fatalf("environment = %s", env)
	}
}
