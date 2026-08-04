package attribution

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/usage"
	"github.com/assaio/assaio/internal/vcs"
)

// epoch is where every scenario's clock starts: a fixed Monday morning, so a fixture is
// reproducible and a scenario states its times as offsets rather than dates.
var epoch = time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)

// mainBranch is the branch a commitSpec lands on when it names none.
const mainBranch = "main"

// Build materializes the scenario into a real repository under dir and reads its commits
// back through the same collector `survival` uses, so the corpus judges an engine against
// the observations it will actually receive rather than a shape invented here.
func (s *Scenario) Build(dir string) (Fixture, error) {
	ctx := context.Background()
	if err := initRepo(ctx, dir); err != nil {
		return Fixture{}, err
	}
	f := Fixture{Root: dir, Hashes: make(map[string]string, len(s.Commits))}
	for i := range s.Commits {
		hash, err := commit(ctx, dir, &s.Commits[i])
		if err != nil {
			return Fixture{}, fmt.Errorf("commit %s: %w", s.Commits[i].Tag, err)
		}
		f.Hashes[s.Commits[i].Tag] = hash
	}
	if err := git(ctx, dir, "checkout", mainBranch); err != nil {
		return Fixture{}, err
	}

	// The collector's window spans the whole fixture: a scenario may commit before its
	// sessions start, and history a scenario built but the corpus cannot see is a fixture
	// that silently tests something narrower than it says.
	commits, skipped, err := vcs.Collect(ctx, dir, epoch.Add(-7*24*time.Hour), epoch, "corpus")
	if err != nil {
		return Fixture{}, err
	}
	if skipped != 0 {
		return Fixture{}, fmt.Errorf("%d commit(s) unreadable: a fixture git cannot describe teaches nothing", skipped)
	}
	f.Commits = commits
	f.Sessions = s.records(vcs.Project(dir))
	f.Confirmed = make(map[string]string, len(s.Corrections))
	for id, tag := range s.Corrections {
		f.Confirmed[id] = f.Hash(tag)
	}
	return f, nil
}

// records turns the session specs into the usage records a store would hold: one at each end
// of the session's window, which is what a time-proximity method has to work from. A session
// with a parent carries Sidechain, the marker the store itself reads to tell sub-agent work
// from ordinary work -- without it a sub-agent scenario is just two unrelated sessions.
func (s *Scenario) records(project string) []usage.Record {
	var out []usage.Record
	for i := range s.Sessions {
		spec := &s.Sessions[i]
		var sidechain int64
		if spec.Parent != "" {
			sidechain = 1
		}
		for turn, offset := range []time.Duration{spec.Start, spec.End} {
			out = append(out, usage.Record{
				Tool:         "claude-code",
				SessionID:    spec.ID,
				Timestamp:    epoch.Add(offset),
				Model:        "claude-sonnet-4-5",
				InputTokens:  100,
				OutputTokens: 200,
				LinesAdded:   spec.Lines,
				DedupeKey:    fmt.Sprintf("%s:%d", spec.ID, turn),
				Project:      project,
				Member:       spec.Member,
				Granularity:  "turn",
				Sidechain:    sidechain,
			})
		}
	}
	return out
}

func initRepo(ctx context.Context, dir string) error {
	for _, args := range [][]string{
		{"init", "--initial-branch=" + mainBranch},
		{"config", "user.email", "corpus@assaio.test"},
		{"config", "user.name", "Corpus"},
	} {
		if err := git(ctx, dir, args...); err != nil {
			return err
		}
	}
	return nil
}

// commit lands one spec on its branch and returns the hash git gave it. Author and commit
// dates are both pinned so the observation's occurrence time is the scenario's, not the
// machine's.
func commit(ctx context.Context, dir string, spec *commitSpec) (string, error) {
	if err := checkout(ctx, dir, spec.Branch); err != nil {
		return "", err
	}
	for _, name := range spec.Files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			return "", err
		}
		if err := os.WriteFile(full, []byte(spec.Tag+"\n"), 0o600); err != nil {
			return "", err
		}
	}
	if err := git(ctx, dir, "add", "."); err != nil {
		return "", err
	}
	stamp := epoch.Add(spec.At).Format(time.RFC3339)
	author := spec.Author
	if author == "" {
		author = "Corpus <corpus@assaio.test>"
	}
	err := gitEnv(ctx, dir, []string{"GIT_AUTHOR_DATE=" + stamp, "GIT_COMMITTER_DATE=" + stamp},
		"commit", "--author="+author, "-m", spec.Tag)
	if err != nil {
		return "", err
	}
	return revParse(ctx, dir)
}

// checkout moves to branch, creating it on first use. An empty branch means the main line.
func checkout(ctx context.Context, dir, branch string) error {
	if branch == "" || branch == mainBranch {
		if err := git(ctx, dir, "rev-parse", "--verify", mainBranch); err != nil {
			return nil // no commit yet: the initial branch is already checked out
		}
		return git(ctx, dir, "checkout", mainBranch)
	}
	if err := git(ctx, dir, "rev-parse", "--verify", branch); err == nil {
		return git(ctx, dir, "checkout", branch)
	}
	return git(ctx, dir, "checkout", "-b", branch)
}

func revParse(ctx context.Context, dir string) (string, error) {
	//nolint:gosec // dir is a caller-provided temp directory and the arguments are literals
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD")
	cmd.Env = isolatedEnv(nil)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func git(ctx context.Context, dir string, args ...string) error {
	return gitEnv(ctx, dir, nil, args...)
}

func gitEnv(ctx context.Context, dir string, env []string, args ...string) error {
	//nolint:gosec // dir is a caller-provided temp directory and args come from the corpus itself
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = isolatedEnv(env)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// isolatedEnv keeps the developer's own git configuration out of the fixture, so a corpus
// run means the same thing on every machine.
func isolatedEnv(extra []string) []string {
	return append(append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null"), extra...)
}
