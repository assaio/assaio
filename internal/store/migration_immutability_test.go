package store

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"testing"
)

// shippedMigrations is every migration that has gone out in a release, with the digest of the
// bytes that shipped. RELEASING.md calls a shipped migration immutable and the runner is why:
// it keys on the filename (schema.go), records each applied name in schema_migration, and skips
// any name it has seen. Editing a shipped file therefore never runs on an upgraded database and
// runs on every fresh one, so the bug is invisible in the author's own testing -- and renaming
// one is worse, because the runner has never seen the new name and re-executes a body it has
// already applied. For 0008 that body moves every claude-code row into an archive table and
// deletes the originals; IF NOT EXISTS guards the DDL and nothing guards the DML.
//
// Adding a migration means adding its digest here in the same commit. Changing a digest that is
// already listed means editing a released migration: write a new numbered file instead.
var shippedMigrations = map[string]string{
	"0001_init.sql":                   "25912eba8bea6328e3745d0d28ff0ba018697eebba4d46387292a63f705d4775",
	"0002_activity_signals.sql":       "33ae906923ca030ab54b802138e05a18262350ef3c8eb69e7aa7cbaf4d84adec",
	"0003_ingest_state.sql":           "55ed2903b42431f42dc6323ce408b9d8b8b1b007e0786f61e8d2ea729fa732b3",
	"0004_ingest_source.sql":          "06eec377ae2d4edf9ada4a5e5e7bd22698d26a91a28108b50cf20f44ca9569c5",
	"0005_session_labels.sql":         "082157372b774533d8a4a80348d19654b5ec48fd0074b3113f1dc841e808f94f",
	"0006_subagent_session_grain.sql": "946959471e9a802f3795646d2904e39e62059f5c6e9e060ffbbe9fcaa86c17f5",
	"0007_cache_tier.sql":             "23b0055c4ff2fbea9f5219114fea342684effa5fe55e8b3eb6f017544958fcc6",
	"0008_response_grain_claude.sql":  "3ac50caee936e78059f2ff43b1a9d26f6406abdbb9634089678681f972ca073e",
	"0009_activity_recount.sql":       "13cf4df8e8fe3f89ac8909e6504325325c8519545209df053cb49dcb48c639fe",
	"0010_duplicate_line_recount.sql": "1a5309b56bfee4cf3e62e5d2a5c0d56c36bf24f7af52fa6cda9a7b453b687800",
	"0011_digest_snapshot.sql":        "05228f8d1d5c2e9f3de40e4334b800241418fa5050fa185cdd6bf480d7d54775",
	"0012_session_step.sql":           "6639b5363c24a15f4055507cbac446988d52cae9d564fa39b358ea4857bfb7c7",
}

// TestShippedMigrationsAreImmutableInNameAndContent is the guard RELEASING.md's hard rule never
// had: no test, no Make target, no CI step and no hook checked it, and the twelve files held
// only because nobody had touched them.
func TestShippedMigrationsAreImmutableInNameAndContent(t *testing.T) {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	onDisk := map[string]bool{}
	for _, e := range entries {
		onDisk[e.Name()] = true
		body, readErr := migrations.ReadFile("migrations/" + e.Name())
		if readErr != nil {
			t.Fatal(readErr)
		}
		sum := sha256.Sum256(body)
		got := hex.EncodeToString(sum[:])
		want, listed := shippedMigrations[e.Name()]
		switch {
		case !listed:
			t.Errorf("migration %s ships with no recorded digest; add %q to shippedMigrations in this commit", e.Name(), got)
		case got != want:
			t.Errorf("migration %s has been edited (%s, want %s). A shipped migration is immutable:"+
				" the runner records its name as applied and skips it, so this SQL will never run on"+
				" an upgraded database and will run on every fresh one. Add a new numbered file instead.",
				e.Name(), got, want)
		}
	}
	for name := range shippedMigrations {
		if !onDisk[name] {
			t.Errorf("migration %s has been renamed or removed. The runner keys on the filename, so a"+
				" rename makes it re-execute a body it has already applied -- for 0008 that is moving"+
				" every claude-code row into the archive table and deleting the originals.", name)
		}
	}
}
