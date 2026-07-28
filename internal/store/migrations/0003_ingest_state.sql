-- Per-input ingest state, so backfill can skip an input it has already parsed unchanged.
-- This table is a cache of what was read; it holds no usage and can be dropped at any
-- time, costing only one slow re-parse.

-- version is the parsing build's identity. A row written by a different build is ignored
-- rather than trusted, which is what keeps store.go's restateSignals repair reachable:
-- an upgraded binary re-parses everything once and history gains the signals the older
-- build could not extract.
CREATE TABLE IF NOT EXISTS ingest_file (
    path        TEXT PRIMARY KEY,
    tool        TEXT NOT NULL,
    size        INTEGER NOT NULL,
    mtime_ns    INTEGER NOT NULL,
    version     TEXT NOT NULL,
    ingested_at TEXT NOT NULL
);
