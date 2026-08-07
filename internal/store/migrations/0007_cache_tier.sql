-- What a cache write bought, and why one could not be read back. Both are reported by
-- Claude Code on every assistant turn and were dropped by the parsers through 0.11.
-- Every column defaults to 0/'' so rows ingested before this migration stay valid and read
-- as "not captured" rather than as a real zero; a re-read of the same local file restates
-- them (internal/store/insert_local.go), which is how history gains the values.

-- The portion of cache_write_tokens that bought a 1-hour cache lifetime rather than the
-- default 5-minute one. A SUBSET of cache_write_tokens, never added to it -- it exists
-- because the vendor bills the two tiers at different rates.
ALTER TABLE usage_record ADD COLUMN cache_write_1h    INTEGER NOT NULL DEFAULT 0;

-- The vendor's own reason this turn's prompt could not be served from cache, from its
-- closed vocabulary (messages_changed, model_changed, previous_message_not_found,
-- system_changed, tools_changed, unavailable). A category label, never content.
ALTER TABLE usage_record ADD COLUMN cache_miss_reason TEXT    NOT NULL DEFAULT '';
