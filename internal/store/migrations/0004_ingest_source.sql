-- Per-source ingest-run history: the baseline the format-drift canaries compare a later
-- run against. Like ingest_file this table is a cache of what was read, holds no usage,
-- and can be dropped at any time.

-- Unlike ingest_file it is bounded by construction: RecordSourceRun keeps only the newest
-- rows per tool and prunes the rest in the same transaction, so this table's size is a
-- function of how many tools are configured, never of how long assaio has been installed.
CREATE TABLE IF NOT EXISTS ingest_source (
    id         INTEGER PRIMARY KEY,
    tool       TEXT    NOT NULL,
    ran_at     TEXT    NOT NULL,
    discovered INTEGER NOT NULL,
    parsed     INTEGER NOT NULL,
    records    INTEGER NOT NULL,
    skipped    INTEGER NOT NULL,
    zero_token INTEGER NOT NULL
);

-- id is a rowid alias, so ordering by it is run order -- which a text timestamp is not,
-- since RFC3339 with trimmed fractional digits does not sort lexicographically.
CREATE INDEX IF NOT EXISTS idx_ingest_source_tool ON ingest_source(tool, id);
