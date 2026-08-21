package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
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
func doctorStore(cmd *cobra.Command, home, dbPath string, since time.Time, window string, maxUnpricedShare float64, traceHorizonDays int) (warnings []drift.Warning, failures []string) {
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, []string{doctorStoreFailure(cmd, "ERROR %v", err)}
	}
	defer func() { _ = st.Close() }()

	n, err := st.Count(cmd.Context())
	if err != nil {
		return nil, []string{doctorStoreFailure(cmd, "ERROR counting records: %v", err)}
	}
	cmd.Printf("store:        ok, %d record(s) at %s\n", n, dbPath)
	if size, sizeErr := st.Size(cmd.Context()); sizeErr == nil {
		cmd.Printf("size:         %s\n", storeSizeLine(size))
	}
	if line := growthLine(cmd.Context(), st, time.Now()); line != "" {
		cmd.Printf("growth:       %s\n", line)
	}
	if h, stepErr := st.Steps(cmd.Context()); stepErr == nil && h.Steps > 0 {
		cmd.Printf("timeline:     %s\n", stepHorizonLine(h))
		cmd.Printf("              %s\n", traceHorizonNote(traceHorizonDays))
	}
	if oldest, histErr := st.HistoryStart(cmd.Context(), "claude-code"); histErr == nil {
		cmd.Printf("retention:    %s\n", retentionLine(home, oldest, time.Now()))
	}
	// The 0.12 upgrade moves the pre-response-grain Claude rows aside rather than deleting
	// them, so the space they hold has to be visible and droppable.
	if rows, archErr := st.LegacyArchiveRows(cmd.Context()); archErr == nil && rows > 0 {
		cmd.Printf("archived:     %d pre-0.12 Claude Code row(s) kept aside by migration 0008; no report reads them.\n", rows)
		cmd.Println("              Drop with: sqlite3 <db> 'DROP TABLE usage_record_pre_response_grain', then `assaio-agent compact`.")
	}
	contents, err := doctorStoreContents(cmd, st, n, since, window)
	if err != nil {
		return nil, []string{doctorStoreFailure(cmd, "ERROR reading stored usage: %v", err)}
	}
	cmd.Printf("inventory:    %s\n", contents.Inventory)
	if failure := unpricedSection(cmd, &contents, maxUnpricedShare); failure != "" {
		failures = append(failures, failure)
	}
	cmd.Printf("freshness:    %s\n", doctorFreshnessLabel(cmd, st))

	warnings, err = driftWarnings(cmd.Context(), st)
	if err != nil {
		return nil, append(failures, doctorStoreFailure(cmd, "ERROR reading ingest history: %v", err))
	}
	cmd.Print(doctorDriftSection(warnings))
	return warnings, failures
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

// traceHorizonNote states the bound on the largest table assaio writes. A zero is honoured
// deliberately and a negative one is rejected by Validate while a lenient load prunes at the
// default, so the three cases read differently.
//
// The remedy names the horizon and nothing else: `clear --older-than` deletes usage records
// first and steps second, so it would trade one table's growth for another's permanent history.
func traceHorizonNote(days int) string {
	switch {
	case days > 0:
		return "pruned past " + strconv.Itoa(days) + " day(s) on every ingest (trace.horizon_days)."
	case days < 0:
		return "trace.horizon_days is negative and was rejected; steps are pruned at the " +
			strconv.Itoa(config.DefaultTraceHorizonDays) + "-day default. Set a valid value."
	default:
		return "trace.horizon_days is 0, so nothing prunes this table and it has no upper bound" +
			" — it grew at a measured 3.40 MB/day on the maintainer's store. Set a horizon to bound it;" +
			" `assaio-agent compact` then returns the freed pages."
	}
}
