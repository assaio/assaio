// Package vcs reads a local git repository and reports what its commits changed, never what
// they changed it to. It is a collector, not a parser: its output is canonical observations
// (ADR 0007), so an analyzer reads commits the same way it reads AI usage and learns nothing
// about where either came from.
//
// The one path-shaped exception is TouchedFiles, which exists so a caller can read the
// working tree; no observation this package emits carries a path, a branch name, a message
// or a diff.
package vcs

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/event"
)

const sourceName = "git"

// The record shape asked of git: a NUL starts a commit header, unit separators divide its
// fields, and the numstat lines that follow belong to it until the next NUL.
const (
	logFormat = "%x00%H%x1f%ct%x1f%P%x1f%s"
	recordSep = "\x00"
	fieldSep  = "\x1f"
)

// revertPrefix is what git itself writes when it generates a revert. An undo phrased any
// other way is invisible to this, which is why the payload calls the field an indicator.
const revertPrefix = `Revert "`

// Collect reads root's commits with a commit date at or after since and returns one
// observation each, newest first, plus the number of commits skipped because their header
// could not be read -- the same skip-and-count policy the log parsers follow, since a git
// version that prints something unexpected must not cost a caller the whole history.
//
// observedAt is passed in rather than read from the clock so one reading time stamps a whole
// pass and the output is a pure function of the repository.
func Collect(ctx context.Context, root string, since, observedAt time.Time, build string) ([]event.Event, int, error) {
	out, err := gitOutput(ctx, root, "log", "--since="+sinceArg(since), "--format="+logFormat, "--numstat")
	if err != nil {
		return nil, 0, err
	}
	var st commitStream
	sc := newScanner(out)
	for sc.Scan() {
		line := sc.Text()
		header, isHeader := strings.CutPrefix(line, recordSep)
		if !isHeader {
			st.count(line)
			continue
		}
		if e, c, ok := headerObservation(header, Project(root), observedAt, build); ok {
			st.start(&e, &c)
		} else {
			st.skip()
		}
	}
	if err := sc.Err(); err != nil {
		return nil, st.skipped, err
	}
	st.close()
	return st.events, st.skipped, nil
}

// commitStream accumulates the observation git is currently describing: a header opens one,
// the numstat lines after it belong to it, and the next header closes it.
type commitStream struct {
	events  []event.Event
	skipped int
	pending *event.Event
	commit  event.Commit
}

func (s *commitStream) start(e *event.Event, c *event.Commit) {
	s.close()
	s.pending, s.commit = e, *c
}

// skip abandons a header this build could not read, closing whatever preceded it.
func (s *commitStream) skip() {
	s.close()
	s.skipped++
}

func (s *commitStream) count(line string) {
	if s.pending != nil {
		countNumstat(&s.commit, line)
	}
}

// close finishes the open observation, keeping it only if the contract vouches for it.
func (s *commitStream) close() {
	if s.pending == nil {
		return
	}
	s.pending.Payload = s.commit
	if s.pending.Validate() == nil {
		s.events = append(s.events, *s.pending)
	} else {
		s.skipped++
	}
	s.pending = nil
}

// headerObservation builds the envelope and the commit shell from one header line. The
// subject is read here to answer one question -- was this a revert -- and is never carried
// onto the payload or returned.
func headerObservation(header, project string, observedAt time.Time, build string) (event.Event, event.Commit, bool) {
	fields := strings.SplitN(header, fieldSep, 4)
	if len(fields) != 4 || fields[0] == "" {
		return event.Event{}, event.Commit{}, false
	}
	seconds, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return event.Event{}, event.Commit{}, false
	}
	return event.Event{
			SpecVersion: event.SpecVersion,
			Type:        event.TypeCommit,
			ID:          fields[0],
			Source:      event.Source{Name: sourceName, Build: build},
			OccurredAt:  time.Unix(seconds, 0).UTC(),
			ObservedAt:  observedAt,
			TimeSource:  event.TimeStated,
			Grain:       event.GrainCommit,
			// Repository evidence stays on the machine until the correlation threat model (B100)
			// decides what a team may share; local-only is the answer that needs no decision.
			Privacy:    event.LocalOnly,
			Provenance: event.Parsed,
			Subject:    event.Subject{Project: project},
		}, event.Commit{
			Parents: int64(len(strings.Fields(fields[2]))),
			Revert:  strings.HasPrefix(fields[3], revertPrefix),
		}, true
}

// countNumstat folds one "added<TAB>removed<TAB>path" line into c. A binary edit, which git
// reports as "-", contributes a changed file and no line counts -- the honest reading, since
// its lines were never counted rather than counted as zero.
func countNumstat(c *event.Commit, line string) {
	fields := strings.SplitN(line, "\t", 3)
	if len(fields) != 3 {
		return
	}
	c.FilesChanged++
	tally(&c.Files, numstatPath(fields[2]))
	if added, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
		c.LinesAdded += added
	}
	if removed, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
		c.LinesRemoved += removed
	}
}
