package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
)

var (
	syncDigestPattern   = regexp.MustCompile(`^member-v2-[0-9a-f]{32}$`)
	legacyMemberPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
)

// SyncIdentityChange reports the rows retained while a server's member keys moved.
type SyncIdentityChange struct {
	Records  int64
	Archived int64
	Labels   int64
	Steps    int64
}

// SyncProtocolVersion is 1 for a store awaiting an explicit identity migration and
// 2 for one allowed to receive branch-free, keyed-member sync records.
func (s *Store) SyncProtocolVersion(ctx context.Context) (int, error) {
	var version int
	err := s.db.QueryRowContext(ctx, `SELECT version FROM sync_protocol WHERE id = 1`).Scan(&version)
	return version, err
}

// MigrateSyncIdentity rekeys every server-side member in one transaction. The caller
// must stop the server and supply a complete, one-to-one old-label to keyed-digest map.
// The old labels are never retained in a mapping table.
func (s *Store) MigrateSyncIdentity(ctx context.Context, mapping map[string]string) (SyncIdentityChange, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SyncIdentityChange{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT version FROM sync_protocol WHERE id = 1`).Scan(&version); err != nil {
		return SyncIdentityChange{}, err
	}
	if version != 1 {
		return SyncIdentityChange{}, errors.New("sync store is already at protocol v2; no identity migration is pending")
	}
	if err := validateSyncMapping(ctx, tx, mapping); err != nil {
		return SyncIdentityChange{}, err
	}
	var change SyncIdentityChange
	for old, digest := range mapping {
		for _, target := range []struct {
			table string
			query string
			count *int64
		}{
			{"usage_record", `UPDATE usage_record SET member = ?,
                dedupe_key = ? || substr(dedupe_key, length(?) + 1), git_branch = ''
                WHERE member = ?`, &change.Records},
			{legacyArchiveTable, `UPDATE usage_record_pre_response_grain SET member = ?,
                dedupe_key = ? || substr(dedupe_key, length(?) + 1), git_branch = ''
                WHERE member = ?`, &change.Archived},
		} {
			result, updateErr := tx.ExecContext(ctx, target.query, digest, digest, old, old)
			if updateErr != nil {
				return SyncIdentityChange{}, fmt.Errorf("rekey %s: %w", target.table, updateErr)
			}
			rows, rowsErr := result.RowsAffected()
			if rowsErr != nil {
				return SyncIdentityChange{}, rowsErr
			}
			*target.count += rows
		}
		for _, target := range []struct {
			table string
			query string
			count *int64
		}{
			{"session_label", `UPDATE session_label SET member = ? WHERE member = ?`, &change.Labels},
			{"session_step", `UPDATE session_step SET member = ? WHERE member = ?`, &change.Steps},
		} {
			result, updateErr := tx.ExecContext(ctx, target.query, digest, old)
			if updateErr != nil {
				return SyncIdentityChange{}, fmt.Errorf("rekey %s: %w", target.table, updateErr)
			}
			rows, rowsErr := result.RowsAffected()
			if rowsErr != nil {
				return SyncIdentityChange{}, rowsErr
			}
			*target.count += rows
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sync_protocol SET version = 2 WHERE id = 1`); err != nil {
		return SyncIdentityChange{}, err
	}
	if err := verifySyncMigration(ctx, tx); err != nil {
		return SyncIdentityChange{}, err
	}
	if err := tx.Commit(); err != nil {
		return SyncIdentityChange{}, err
	}
	return change, nil
}

func validateSyncMapping(ctx context.Context, tx *sql.Tx, mapping map[string]string) error {
	if len(mapping) == 0 {
		return errors.New("identity map is empty")
	}
	seen := make(map[string]string, len(mapping))
	for old, digest := range mapping {
		if !legacyMemberPattern.MatchString(old) {
			return fmt.Errorf("invalid old member %q", old)
		}
		if !syncDigestPattern.MatchString(digest) {
			return fmt.Errorf("member %q has invalid v2 digest", old)
		}
		if previous, duplicate := seen[digest]; duplicate {
			return fmt.Errorf("members %q and %q map to the same digest", previous, old)
		}
		seen[digest] = old
	}
	rows, err := tx.QueryContext(ctx, `
        SELECT member FROM usage_record WHERE member <> ''
        UNION SELECT member FROM usage_record_pre_response_grain WHERE member <> ''
        UNION SELECT member FROM session_label WHERE member <> ''
        UNION SELECT member FROM session_step WHERE member <> ''`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	existing := make(map[string]bool)
	for rows.Next() {
		var member string
		if err := rows.Scan(&member); err != nil {
			return err
		}
		existing[member] = true
		if _, ok := mapping[member]; !ok {
			return fmt.Errorf("no v2 digest for existing member %q", member)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for old, digest := range mapping {
		if !existing[old] {
			return fmt.Errorf("mapped member %q has no stored rows", old)
		}
		if existing[digest] {
			return fmt.Errorf("target digest %q already owns rows", digest)
		}
	}
	for _, table := range []string{"usage_record", legacyArchiveTable} {
		query := `SELECT COUNT(*) FROM ` + table + ` WHERE member <> '' AND
            substr(dedupe_key, 1, length(member) + 1) <> member || ':'`
		var malformed int64
		if err := tx.QueryRowContext(ctx, query).Scan(&malformed); err != nil {
			return err
		}
		if malformed != 0 {
			return fmt.Errorf("%s has %d member rows without their dedupe-key prefix", table, malformed)
		}
	}
	return nil
}

func verifySyncMigration(ctx context.Context, tx *sql.Tx) error {
	for _, table := range []string{"usage_record", legacyArchiveTable} {
		query := `SELECT COUNT(*) FROM ` + table + ` WHERE member <> '' AND
            (member NOT GLOB 'member-v2-*' OR git_branch <> '')`
		var invalid int64
		if err := tx.QueryRowContext(ctx, query).Scan(&invalid); err != nil {
			return err
		}
		if invalid != 0 {
			return fmt.Errorf("%s retained %d legacy identities or branches", table, invalid)
		}
	}
	return nil
}
