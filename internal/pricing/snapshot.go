package pricing

// SnapshotDate is the date the vendored litellm.json was downloaded; update when refreshing the file.
const SnapshotDate = "2026-07-11"

// Info loads the embedded price table and returns its model count and snapshot date.
func Info() (models int, snapshotDate string) {
	t, err := Load()
	if err != nil {
		return 0, SnapshotDate
	}
	return len(t), SnapshotDate
}
