-- project is the basename of a repository root, which two unrelated repositories can share. These
-- tables record which repository a row came from, apart from the name it carries.
--
-- repository holds each identity once: the label, an opaque local key for the root
-- (internal/projectid.Key -- a keyed hash of the root's path, never exported), and the ordinal
-- that orders repositories sharing a label. Ordinals are assigned max+1 and never reused.
CREATE TABLE IF NOT EXISTS repository (
    id       INTEGER PRIMARY KEY,
    label    TEXT    NOT NULL CHECK (label <> ''),
    repo_key TEXT    NOT NULL CHECK (repo_key <> ''),
    ordinal  INTEGER NOT NULL CHECK (ordinal > 0),
    UNIQUE (label, repo_key),
    UNIQUE (label, ordinal)
);

-- The salt the keys are derived with. It lives in the store rather than beside it: the store is
-- a one-file backup, and a salt kept elsewhere would re-key every repository after a restore.
CREATE TABLE IF NOT EXISTS repository_salt (
    id   INTEGER PRIMARY KEY CHECK (id = 1),
    salt BLOB    NOT NULL CHECK (length(salt) = 32)
);
INSERT OR IGNORE INTO repository_salt (id, salt) VALUES (1, randomblob(32));

-- A store whose archive was dropped (DropLegacyArchive) gets it back empty before either ALTER,
-- as 0013 does; recreating it after the first ALTER would copy the new column and fail the second.
CREATE TABLE IF NOT EXISTS usage_record_pre_response_grain AS
    SELECT * FROM usage_record WHERE 0;

-- repo_id is 0 for a row whose repository is unresolved: no working directory in the log, no
-- repository above it, a directory that was gone when read, a record from sync or a plugin, or a
-- row stored before this migration. -1 is ambiguous: two reads of the row named different
-- repositories, and it stays ambiguous. SQLite adds a constant-default column without rewriting
-- existing rows.
ALTER TABLE usage_record
    ADD COLUMN repo_id INTEGER NOT NULL DEFAULT 0 CHECK (repo_id >= -1);

ALTER TABLE usage_record_pre_response_grain
    ADD COLUMN repo_id INTEGER NOT NULL DEFAULT 0 CHECK (repo_id >= -1);

-- A repository names rows only while some row carries its id; the lookup that decides it. Partial,
-- so an upgraded store, where every row is 0, does not grow until rows resolve.
CREATE INDEX IF NOT EXISTS idx_usage_repo ON usage_record(repo_id) WHERE repo_id > 0;

-- Existing rows have no repository. Clearing the watermarks of the sources that record a working
-- directory makes the next backfill re-read every such transcript still on disk, and the re-read
-- restates the row's repository; a source build stamps every read 'dev', so without the clear it
-- would skip each transcript as unchanged.
DELETE FROM ingest_file WHERE tool IN ('claude-code', 'codex', 'copilot-cli');
