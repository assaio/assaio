package store

import "context"

// legacyArchiveTable holds the Claude Code rows migration 0008 moved out of usage_record:
// usage the pre-0.12 parser stored once per content block instead of once per API response.
// Nothing queries it -- it exists so an upgrade never destroys history the user's own
// transcripts may no longer be able to rebuild, since Claude Code rotates its logs.
const legacyArchiveTable = "usage_record_pre_response_grain"

// LegacyArchiveRows reports how many pre-0.12 rows the archive still holds. Zero covers both
// states a caller treats identically: a fresh install, which has no history to keep, and an
// archive already dropped. Callers use this to tell the user what is being kept and what
// drops it, so the space it holds is a decision rather than a surprise.
func (s *Store) LegacyArchiveRows(ctx context.Context) (int64, error) {
	var name string
	if err := s.db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
		legacyArchiveTable).Scan(&name); err != nil {
		// No row means no archive, which is the ordinary state, not a failure.
		return 0, nil
	}
	var rows int64
	// The identifier is this package's own constant, never caller input.
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+legacyArchiveTable).Scan(&rows); err != nil {
		return 0, err
	}
	return rows, nil
}

// DropLegacyArchive removes the archive. The freed pages stay in the file until Vacuum, so
// callers pair it with compact.
func (s *Store) DropLegacyArchive(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DROP TABLE IF EXISTS `+legacyArchiveTable)
	return err
}
