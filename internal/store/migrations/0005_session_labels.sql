-- Session annotations: the intent layer the logs do not contain. Unlike every other table
-- here this one holds data a person typed, so it cannot be rebuilt by re-reading the logs --
-- which is why `clear` leaves it alone and only `clear --labels` deletes it.
--
-- Category labels only (internal/label holds the vocabularies), so the table is free of
-- content by construction: there is nowhere to put a branch name, a ticket title, or a
-- prompt. Deliberately no CHECK constraint -- Go validates, so adding a category stays a
-- one-line change rather than a migration.
--
-- Bounded by hand, not by history: one row per session someone marked, ~80 bytes. A store
-- with a thousand annotations carries ~80 KB, and that figure does not grow with the volume
-- of logs ingested.
--
-- The primary key is (session_id, member) -- the same key store.Sessions groups by -- so a
-- central store cannot blend two members' annotations into one.
CREATE TABLE IF NOT EXISTS session_label (
    session_id TEXT NOT NULL,
    member     TEXT NOT NULL DEFAULT '',
    task       TEXT NOT NULL DEFAULT '',
    outcome    TEXT NOT NULL DEFAULT '',
    difficulty TEXT NOT NULL DEFAULT '',
    marked_at  TEXT NOT NULL,
    PRIMARY KEY (session_id, member)
);

-- B70: the composite index every project-scoped window query wants. idx_usage_ts alone
-- makes a per-project drill scan the whole window and filter row by row.
CREATE INDEX IF NOT EXISTS idx_usage_project_ts ON usage_record(project, ts);
