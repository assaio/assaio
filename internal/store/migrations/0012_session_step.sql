-- The session as a sequence rather than a total. One row per step: what the agent did, in
-- what order, at what token cost, and how it ended.
--
-- Content-free by construction. No prompt, no code, no file name, no path. target_ref is an
-- integer assigned in first-seen order within one session and nothing else -- deliberately not
-- a digest of the path, because a digest is reversible by anyone holding the repository, paths
-- carrying almost no entropy. "The same target nine times" stays answerable; "which file" does
-- not.
--
-- Size, measured on the maintainer's store after a full rebuild under the 30-day horizon:
-- 335,527 steps against 178,016 usage records (1.88 stored steps per record), and 101.9 MB of
-- table and indexes against usage_record's 58.3 MB. The timeline is roughly 1.7x the usage
-- table it describes.
--
-- The row multiplier alone is not a size measurement -- bytes are dominated by per-row overhead
-- and indexes, not column count -- which is why B147's "a different size class" was right and
-- why the horizon is load-bearing rather than tidy. See Store.PruneSteps.
CREATE TABLE IF NOT EXISTS session_step (
    id         INTEGER PRIMARY KEY,
    tool       TEXT    NOT NULL,
    session_id TEXT    NOT NULL,
    -- timeline separates the sequences that share a session: '' is the main loop, anything
    -- else is the id of a sub-agent whose transcript records its parent's session_id. Without
    -- it, ordinal is not unique within a session -- the worst session in the maintainer's
    -- store had 222 steps at position 1, one per sub-agent transcript.
    timeline   TEXT    NOT NULL DEFAULT '',
    dedupe_key TEXT    NOT NULL,
    ts         TEXT    NOT NULL,
    -- ordinal is the step's position within (session_id, member, timeline), from 1. It orders
    -- what a timestamp cannot: a turn and its tool calls are frequently stamped the same second.
    -- A sequence may legitimately start above 1: the horizon cuts the opening off a timeline
    -- that straddles it, which a reader has to allow for rather than read as data loss.
    ordinal    INTEGER NOT NULL,
    kind       TEXT    NOT NULL,
    -- outcome is '' when the source did not say, which is a different fact from 'ok'.
    outcome    TEXT    NOT NULL DEFAULT '',
    model      TEXT    NOT NULL DEFAULT '',
    tokens     INTEGER NOT NULL DEFAULT 0,
    target_ref INTEGER NOT NULL DEFAULT 0,
    -- member mirrors usage_record's: '' for local rows, set only by a sync push. A session id
    -- is unique per member, so every read correlates on the pair.
    member     TEXT    NOT NULL DEFAULT '',
    -- The timeline belongs in the key: a forked sub-agent replays its origin's whole prefix
    -- under a new agentId, so the same message.id and tool_use.id legitimately appear in more
    -- than one sequence (141 of them across a 400-file sample; one id in three timelines).
    -- Keyed on (tool, dedupe_key) alone, INSERT OR IGNORE kept whichever file ingest reached
    -- first and 13,897 steps vanished across 43 sequences, one of them starting at ordinal
    -- 3,637 with no way to know its opening was missing.
    UNIQUE(tool, timeline, dedupe_key)
);

CREATE INDEX IF NOT EXISTS idx_step_session
    ON session_step(session_id, member, timeline, ordinal);
CREATE INDEX IF NOT EXISTS idx_step_ts ON session_step(ts);
