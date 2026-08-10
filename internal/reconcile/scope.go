package reconcile

import "time"

// Scope is how far the two sides fail to describe the same thing. It is computed before any
// delta and every later figure is restricted to Overlap, because an export window almost
// never equals the queried window: comparing their totals directly reports a difference in
// date ranges as a difference in money.
type Scope struct {
	ExportFirst string `json:"export_first"`
	ExportLast  string `json:"export_last"`
	WindowFirst string `json:"window_first"`
	WindowLast  string `json:"window_last"`
	// OverlapDays is how many dates carry data on at least one side inside both ranges.
	OverlapDays int `json:"overlap_days"`
	// ExportOnlyDays and WindowOnlyDays count dates each side covers alone.
	ExportOnlyDays int `json:"export_only_days"`
	WindowOnlyDays int `json:"window_only_days"`
	// ExportCostOutside is money the export bills on dates outside the overlap, and
	// EstimateCostOutside the same for the local estimate. Both are excluded from the
	// comparison and reported so the exclusion is visible rather than silent.
	ExportCostOutside   float64 `json:"export_cost_outside"`
	EstimateCostOutside float64 `json:"estimate_cost_outside"`

	overlap map[string]bool
}

// Overlap reports whether d is one of the dates the comparison runs over.
func (s *Scope) Overlap(d string) bool { return s.overlap[d] }

// Aligned reports whether the two sides describe exactly the same dates.
func (s *Scope) Aligned() bool { return s.ExportOnlyDays == 0 && s.WindowOnlyDays == 0 }

// buildScope intersects the export's dates with the queried window. The window bound is the
// requested range, not the dates the store happens to hold: a day inside the window with no
// local usage is a real zero on the estimate side, and dropping it would hide a day the
// vendor billed for and assaio saw nothing on.
func buildScope(exp *Export, estimateDays map[string]float64, start, end time.Time) Scope {
	s := Scope{WindowFirst: day(start), WindowLast: day(end), overlap: map[string]bool{}}
	s.ExportFirst, s.ExportLast = exp.Span()

	for _, r := range exp.Rows {
		switch {
		case inWindow(r.Day, s.WindowFirst, s.WindowLast):
			s.overlap[r.Day] = true
		default:
			s.ExportCostOutside += r.Cost
		}
	}
	exportDays := exp.Days()
	for d, cost := range estimateDays {
		if !inWindow(d, s.WindowFirst, s.WindowLast) {
			s.EstimateCostOutside += cost
			continue
		}
		if !exportDays[d] && !withinSpan(d, s.ExportFirst, s.ExportLast) {
			// The export cannot speak for a date its own span never reached.
			s.EstimateCostOutside += cost
			continue
		}
		s.overlap[d] = true
	}

	for d := range exportDays {
		if !s.overlap[d] {
			s.ExportOnlyDays++
		}
	}
	for d := range estimateDays {
		if !s.overlap[d] {
			s.WindowOnlyDays++
		}
	}
	s.OverlapDays = len(s.overlap)
	return s
}

func inWindow(d, first, last string) bool { return d >= first && d <= last }

// withinSpan reports whether d falls inside the export's own first..last range, which is
// what lets a zero-cost day inside a covered span count as covered.
func withinSpan(d, first, last string) bool {
	if first == "" || last == "" {
		return false
	}
	return d >= first && d <= last
}
