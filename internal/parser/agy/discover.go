package agy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/assaio/assaio/internal/usage"
)

// transcriptPath is the conversation-relative location of the log this parser reads. The
// sibling transcript_full.jsonl holds the same entries with the vendor's truncation undone,
// so it is larger and carries strictly more prompt text for no additional accounting.
var transcriptPath = filepath.Join(".system_generated", "logs", "transcript.jsonl")

// Discover returns the conversation directories under one Antigravity CLI root. The glob is
// narrow for the reason Gemini CLI's is: ~/.gemini is shared, and ~/.gemini/antigravity-cli
// holds an OAuth token, a settings file and a per-conversation SQLite database beside the
// transcripts. Only a directory that actually holds a transcript is returned.
func Discover(root string) ([]string, error) {
	found, err := filepath.Glob(filepath.Join(root, "brain", "*", transcriptPath))
	if err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(found))
	for _, f := range found {
		dirs = append(dirs, strings.TrimSuffix(f, string(filepath.Separator)+transcriptPath))
	}
	return dirs, nil
}

// ParseDir reads one conversation directory as Discover returns it. The conversation id is
// the directory's own name, which is where the vendor keeps it: nothing inside the transcript
// identifies the conversation.
func ParseDir(dir string) ([]usage.Record, int, error) {
	f, err := os.Open(filepath.Join(dir, transcriptPath)) //nolint:gosec // a path from this package's own discovery glob
	if err != nil {
		// os.Open fails with a *fs.PathError, whose message is the whole path -- here a
		// person's home, the conversation's uuid and all. Only the reason travels: what could
		// not be opened is already named, and PRIVACY.md keeps a local path out of anything
		// assaio prints or stores. Unwrapped rather than formatted away so errors.Is still
		// reaches fs.ErrNotExist.
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			err = pathErr.Err
		}
		return nil, 0, fmt.Errorf("open agy transcript: %w", err)
	}
	defer func() { _ = f.Close() }()
	return ParseTranscript(f, filepath.Base(dir))
}
