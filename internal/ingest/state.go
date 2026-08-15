package ingest

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/version"
)

// devVersion is the identity a locally built binary carries. It is deliberately stable
// across rebuilds, so parser work needs `backfill --full` to be re-read; mixing in the VCS
// revision would instead impose a full re-parse on every commit.
const devVersion = "dev"

// Options tunes one ingest run. Full re-parses every input regardless of stored state.
type Options struct {
	Full bool
	// TraceHorizonDays bounds how far back the step timeline is kept; steps older than it are
	// pruned at the end of a run. 0 means no horizon at all -- keep every step, prune nothing --
	// which is config.Trace.HorizonDays' own deliberate semantics and the setting that makes the
	// table unbounded (see internal/config/trace.go and traceHorizon below).
	TraceHorizonDays int
}

// input is one discovered file or Cline task directory, carrying the signature that
// decides whether it still needs parsing. statErr is kept rather than raised: an input
// assaio cannot stat is always parsed, so the real error surfaces at the parse site with
// the file's name on it.
type input struct {
	path, tool    string
	size, mtimeNS int64
	statErr       error
}

// fileInput signs one session file by its size and modification time.
func fileInput(path, tool string) input {
	in := input{path: path, tool: tool}
	fi, err := os.Stat(path)
	if err != nil {
		in.statErr = err
		return in
	}
	in.size, in.mtimeNS = fi.Size(), fi.ModTime().UnixNano()
	return in
}

// dirInput signs a Cline task directory by the total size and the newest modification
// time inside it, so a change to any file it contains invalidates the whole directory.
func dirInput(dir, tool string) input {
	in := input{path: dir, tool: tool}
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		in.size += fi.Size()
		if ns := fi.ModTime().UnixNano(); ns > in.mtimeNS {
			in.mtimeNS = ns
		}
		return nil
	})
	if err != nil {
		in.statErr = err
	}
	return in
}

// skipper answers which inputs this build has already parsed, and collects the ones it
// parses now. An input is recorded only after a clean parse, so a file that fails keeps
// being retried and keeps appearing in the Failed count instead of silently vanishing
// from it on the next run.
type skipper struct {
	known    map[string]store.IngestEntry
	full     bool
	ingested []store.IngestEntry
}

// newSkipper loads the state written by this build. Full skips the load entirely: nothing
// is known, so every input is parsed.
func newSkipper(ctx context.Context, st *store.Store, full bool) (*skipper, error) {
	if full {
		return &skipper{full: true}, nil
	}
	known, err := st.IngestState(ctx, buildIdentity())
	if err != nil {
		return nil, err
	}
	return &skipper{known: known}, nil
}

// skip reports whether in is unchanged since this build last parsed it.
func (s *skipper) skip(in input) bool {
	if s.full || in.statErr != nil {
		return false
	}
	e, ok := s.known[in.path]
	return ok && e.Size == in.size && e.MtimeNS == in.mtimeNS
}

// record marks in as cleanly parsed by this build.
func (s *skipper) record(in input) {
	s.ingested = append(s.ingested, store.IngestEntry{
		Path: in.path, Tool: in.tool, Size: in.size, MtimeNS: in.mtimeNS,
	})
}

// flush persists everything parsed in this run.
func (s *skipper) flush(ctx context.Context, st *store.Store, at time.Time) error {
	return st.RecordIngest(ctx, buildIdentity(), at, s.ingested)
}

// buildIdentity is the parsing build's identity, stamped on every ingest_file row and
// required to match before an input is skipped. Keying on it is what keeps
// store.Insert's restateSignals repair reachable: a new build sees no usable state and
// re-parses everything once, so history gains signals the older parser could not extract.
// Release binaries carry the ldflags version; `go install ...@vX.Y.Z` falls back to the
// module version, which is the only identity a source install has.
func buildIdentity() string {
	if v := version.Version; v != "" && v != devVersion {
		return v
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return devVersion
}
