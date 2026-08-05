-- A completed Claude sub-agent is stored as one record totalling a whole sub-agent run, and
-- the parser labelled it 'turn' until v0.10. That is the exact case the granularity rule
-- exists for: a summary of many requests must never be read as one of them, or every
-- per-turn figure counts it as a single very large turn.
--
-- A re-parse cannot fix these rows. Once a sub-agent has its own transcript file the parent's
-- aggregate is suppressed at parse time, so the row already written is never offered to the
-- store again -- 1,015 of them on the maintainer's machine after a full backfill.
--
-- The predicate is store.Delegation's, exactly: 'agent:<id>' is what the Claude parser writes
-- locally, and '<member>:agent:<id>' is what the sync server rewrites it to. Scoping to
-- claude-code keeps another tool's key scheme from matching by coincidence. No row is added
-- and no column is widened, so the store does not grow and there is nothing to clean up.
UPDATE usage_record
   SET granularity = 'session'
 WHERE tool = 'claude-code'
   AND (dedupe_key LIKE 'agent:%' OR dedupe_key LIKE '%:agent:%');
