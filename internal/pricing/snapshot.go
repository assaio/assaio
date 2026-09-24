package pricing

// SnapshotDate is the date the vendored litellm.json was downloaded; `make prices` sets it.
const SnapshotDate = "2026-09-23"

// Info reports the embedded table: how many keys the LiteLLM snapshot prices, how many more
// only retained.json prices because a later snapshot dropped them, and the snapshot date.
func Info() (models, retained int, snapshotDate string, err error) {
	t, err := Load()
	if err != nil {
		return 0, 0, SnapshotDate, err
	}
	return len(t) - cachedRetained, cachedRetained, SnapshotDate, nil
}
