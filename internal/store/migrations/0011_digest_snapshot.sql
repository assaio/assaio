-- v0.17 adds `digest --weekly`, which reports what *moved* rather than restating what is.
-- Movement is a comparison, and every aggregate in this store reads `ts >= since` with no
-- closed upper bound -- so there is nothing to compare a window against. Each digest
-- therefore records what it reported, and the next one reads it.
--
-- That also makes the comparison honest about itself. The snapshot carries the build that
-- parsed the data behind it, so a digest whose reader changed since the last run can say
-- so instead of reporting a mover that is really a parser fix correcting history.
--
-- Bounded by construction: writing a snapshot deletes all but the newest few, so this table
-- does not grow with time. A snapshot is a few KB of verdicts and totals -- never usage rows,
-- never anything a person typed.
CREATE TABLE IF NOT EXISTS digest_snapshot (
    taken_at  TEXT PRIMARY KEY,
    parsed_by TEXT NOT NULL DEFAULT '',
    window    TEXT NOT NULL DEFAULT '',
    payload   TEXT NOT NULL
);
