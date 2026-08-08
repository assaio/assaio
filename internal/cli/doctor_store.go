package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/drift"
	"github.com/assaio/assaio/internal/store"
)

// doctorStore prints everything doctor reads out of the store -- health, size, inventory,
// freshness and the drift canaries -- and reports what a --strict run must fail on.
//
// A store it cannot read is a finding, not a reason to stop: returning early there printed
// ERROR and exited 0, so a cron job with a corrupt database reported green while the
// canaries the whole --strict promise rests on never ran. The failure travels back as a
// string so the caller can weigh it beside the other strict failures instead of deciding
// doctor's exit code halfway down the report.
func doctorStore(cmd *cobra.Command, dbPath string) (warnings []drift.Warning, failure string) {
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, doctorStoreFailure(cmd, "ERROR %v", err)
	}
	defer func() { _ = st.Close() }()

	n, err := st.Count(cmd.Context())
	if err != nil {
		return nil, doctorStoreFailure(cmd, "ERROR counting records: %v", err)
	}
	cmd.Printf("store:        ok, %d record(s) at %s\n", n, dbPath)
	if size, sizeErr := st.Size(cmd.Context()); sizeErr == nil {
		cmd.Printf("size:         %s\n", storeSizeLine(size))
	}
	// The 0.12 upgrade moves the pre-response-grain Claude rows aside rather than deleting
	// them, so the space they hold has to be visible and droppable.
	if rows, archErr := st.LegacyArchiveRows(cmd.Context()); archErr == nil && rows > 0 {
		cmd.Printf("archived:     %d pre-0.12 Claude Code row(s) kept aside by migration 0008; no report reads them.\n", rows)
		cmd.Println("              Drop with: sqlite3 <db> 'DROP TABLE usage_record_pre_response_grain', then `assaio-agent compact`.")
	}
	cmd.Printf("inventory:    %s\n", doctorInventoryLabel(cmd, st, n))
	cmd.Printf("freshness:    %s\n", doctorFreshnessLabel(cmd, st))

	warnings, err = driftWarnings(cmd.Context(), st)
	if err != nil {
		return nil, doctorStoreFailure(cmd, "ERROR reading ingest history: %v", err)
	}
	cmd.Print(doctorDriftSection(warnings))
	return warnings, ""
}

// doctorStoreFailure prints the store line and the drift line that go with a store nobody
// could read, and returns the sentence --strict fails on. Saying the canaries did not run
// matters as much as the error: "no canary fired" would read as a clean bill of health.
func doctorStoreFailure(cmd *cobra.Command, format string, args ...any) string {
	detail := fmt.Sprintf(format, args...)
	cmd.Printf("store:        %s\n", detail)
	cmd.Println("drift:        not evaluated — the canaries need a readable store")
	return "store: " + detail
}
