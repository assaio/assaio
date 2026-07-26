-- Activity signals present in the tools' own logs but dropped by the 0.1/0.2 parsers.
-- Each column exists because a shipped validator reads it; nothing here is speculative.
-- Every column defaults to 0/'' so rows ingested before this migration stay valid and
-- simply read as "not captured" rather than as a real zero.

-- Tool-call taxonomy: the same turn's tool calls split by what they were for, so a window
-- can be read as exploring (reading/searching) versus producing (writing code).
ALTER TABLE usage_record ADD COLUMN tool_reads    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_record ADD COLUMN tool_searches INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_record ADD COLUMN tool_commands INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_record ADD COLUMN tool_writes   INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_record ADD COLUMN tool_other    INTEGER NOT NULL DEFAULT 0;

-- Tool calls that came back an error: a friction signal distinct from a human rejection.
ALTER TABLE usage_record ADD COLUMN tool_errors   INTEGER NOT NULL DEFAULT 0;

-- 1 when this turn ran inside a sub-agent, read from the log itself rather than inferred
-- from the dedupe key.
ALTER TABLE usage_record ADD COLUMN sidechain     INTEGER NOT NULL DEFAULT 0;

-- Which skill and which sub-agent type the turn is attributed to, as the tool labelled it.
-- Category labels only -- never a prompt, a path, or any content.
ALTER TABLE usage_record ADD COLUMN skill         TEXT    NOT NULL DEFAULT '';
ALTER TABLE usage_record ADD COLUMN agent         TEXT    NOT NULL DEFAULT '';
