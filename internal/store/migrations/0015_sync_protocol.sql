-- A store with existing synced rows needs an explicit identity rekey before v2 can write.
-- Fresh stores may accept v2 immediately. The rekey moves this marker in the same
-- transaction as the rows, so a crash cannot expose a half-migrated store.
CREATE TABLE IF NOT EXISTS usage_record_pre_response_grain AS
    SELECT * FROM usage_record WHERE 0;
CREATE TABLE IF NOT EXISTS sync_protocol (
    id      INTEGER PRIMARY KEY CHECK (id = 1),
    version INTEGER NOT NULL CHECK (version IN (1, 2))
);
INSERT INTO sync_protocol (id, version)
SELECT 1, CASE WHEN
    EXISTS (SELECT 1 FROM usage_record WHERE member <> '') OR
    EXISTS (SELECT 1 FROM usage_record_pre_response_grain WHERE member <> '') OR
    EXISTS (SELECT 1 FROM session_label WHERE member <> '') OR
    EXISTS (SELECT 1 FROM session_step WHERE member <> '')
    THEN 1 ELSE 2 END;
