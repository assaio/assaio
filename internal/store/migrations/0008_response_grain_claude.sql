-- Claude Code writes one JSONL line per content block of a single API response, and every
-- one of those lines repeats that response's usage verbatim. Through 0.11 the parser keyed a
-- record on the line's uuid, so one request's tokens were stored once per block: measured on
-- 5,724 real transcripts, 354,904 assistant lines were 159,175 responses, inflating output
-- tokens 1.97x and cache-write tokens 2.81x.

-- The parser now keys on the response id, so a re-read produces different keys and would land
-- beside the inflated rows rather than replacing them. They therefore have to leave
-- usage_record -- but they are NOT destroyed, because the rebuild is only as deep as the
-- transcripts that still exist and Claude Code rotates its own logs (30 days by default).
-- The store, not the transcript directory, is the durable record, and this migration runs
-- from store.Open -- which the statusline calls -- so it would otherwise fire on the first
-- prompt after the binary is replaced, before anyone had read a changelog or copied the file.
-- The rows are moved aside instead: wrong enough that no report may read them, real enough
-- that no upgrade may delete them silently.
-- Created empty and filled by a separate INSERT rather than CREATE ... AS SELECT: the latter
-- is a silent no-op when the table already exists, which would move the rows out of
-- usage_record without ever copying them. A fresh install runs this too and is left with an
-- empty archive, which is the honest shape -- it has no pre-0.12 history to keep.
CREATE TABLE IF NOT EXISTS usage_record_pre_response_grain AS
    SELECT * FROM usage_record WHERE 0;

INSERT INTO usage_record_pre_response_grain
    SELECT * FROM usage_record WHERE tool = 'claude-code';

DELETE FROM usage_record WHERE tool = 'claude-code';

-- The rebuild has to be unconditional, so the watermarks go with the rows. Relying on
-- ingest_file's version stamp changing is not enough: buildIdentity() returns the constant
-- "dev" for any source build, so a plain `backfill` would find every transcript unchanged and
-- leave the history empty until someone happened to pass --full.
DELETE FROM ingest_file WHERE tool = 'claude-code';

-- The archive is a one-off, bounded by whatever the old parser had already stored, and
-- nothing queries it. `assaio-agent doctor` reports its size and the statement that drops it;
-- `assaio-agent compact` returns the pages afterwards.
