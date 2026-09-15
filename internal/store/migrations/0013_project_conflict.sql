-- A source can repeat one record with different repository working directories. The record's
-- counts still exist once, but neither project is proven: keeping whichever file was parsed
-- last makes input order decide attribution. This bit is sticky so a later partial re-read
-- cannot turn an observed conflict back into a project claim.
--
-- SQLite adds a constant-default column by changing the table schema, without rewriting the
-- existing usage rows. Existing records start unconflicted; a full re-import can discover a
-- disagreement that an older build collapsed.
CREATE TABLE IF NOT EXISTS usage_record_pre_response_grain AS
    SELECT * FROM usage_record WHERE 0;

ALTER TABLE usage_record
    ADD COLUMN project_conflict INTEGER NOT NULL DEFAULT 0
        CHECK (project_conflict IN (0, 1));

-- Migration 0008 copies rows with SELECT * when upgrading a pre-v0.12 store. Its archive must
-- keep the same shape if that migration is recovered from a missing watermark later. A user who
-- already dropped the archive gets an empty table; LegacyArchiveRows treats that as absent.
ALTER TABLE usage_record_pre_response_grain
    ADD COLUMN project_conflict INTEGER NOT NULL DEFAULT 0
        CHECK (project_conflict IN (0, 1));
