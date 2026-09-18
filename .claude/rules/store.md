---
paths:
  - "internal/store/**"
  - "internal/paths/**"
---

# The store

- **A shipped migration is immutable in name and content.** Every schema change is a new
  `internal/store/migrations/000N_*.sql`; `TestShippedMigrationsAreImmutableInNameAndContent`
  carries a digest per file and a new migration adds its digest in the same commit. A rename
  re-runs applied DML (`RELEASING.md`, "Schema changes").
- **Size is measured in bytes on a real corpus** (`Size()`, `dbstat`), never estimated from rows;
  every growth ships a retention rule enforced on ingest (`trace.horizon_days` is the model) and
  the note that SQLite never shrinks on DELETE (`compact`).
- **Labels are the only rows no re-import rebuilds**; a deletion path that could take them says so.
- **A correction must reach history**: say whether rows written under the old rule are rebuilt by
  `backfill --full`, corrected by a restate path, or unreachable — and a `MAX`-style restate cannot
  lower a figure (`B116`).
- **The real store is `~/.local/share/assaio/assaio.db`** (170 MB, days the sources deleted).
  `backfill`, `clear`, `compact`, `serve`, `statusline` have no `--db`: every local run sets
  `XDG_DATA_HOME=$(mktemp -d)`. The hook denies `clear` without it.
- Claude Code deletes transcripts after 30 days; history beyond it exists only here.
