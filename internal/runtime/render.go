package runtime

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Experimental is the label every surface of this feature carries. It is not decoration: the
// command exists to find out whether combined usage and runtime evidence changes a decision,
// and until three self-hosted deployments say it does, nothing here is a promise.
const Experimental = "experimental"

// RenderText writes one snapshot as a block: what was read, when, what it found, what it did
// not, and the limits that travel with a single read. The order is deliberate -- availability
// first, then present capabilities, then missing ones -- because an inspection that listed only
// what it found would read as complete.
func RenderText(w io.Writer, s *Snapshot) error {
	if _, err := fmt.Fprintf(w, "%s · %s  [%s]\n", strings.ToUpper(s.Source), s.Availability, Experimental); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  read from %s at %s\n", s.Origin, s.ObservedAt.UTC().Format("2006-01-02T15:04:05Z")); err != nil {
		return err
	}
	if s.Error != "" {
		if _, err := fmt.Fprintf(w, "  could not read it: %s\n", s.Error); err != nil {
			return err
		}
	}
	if err := renderIntegrity(w, s); err != nil {
		return err
	}
	if err := renderPresent(w, s); err != nil {
		return err
	}
	return renderMissing(w, s)
}

// renderIntegrity states what would make every absence below unproven, before the absences are
// listed rather than after.
func renderIntegrity(w io.Writer, s *Snapshot) error {
	if s.SkippedLines == 0 && !s.Truncated {
		return nil
	}
	if s.SkippedLines > 0 {
		if _, err := fmt.Fprintf(w, "  %d line(s) could not be parsed and were skipped.\n", s.SkippedLines); err != nil {
			return err
		}
	}
	if s.Truncated {
		if _, err := fmt.Fprintln(w, "  A size limit stopped this read before the document ended, so anything reported missing below may simply be past the cut."); err != nil {
			return err
		}
	}
	return nil
}

func renderPresent(w io.Writer, s *Snapshot) error {
	present := s.Present()
	if len(present) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(w, "\n  found %d of %d metric families assaio reads:\n", len(present), len(s.Findings)); err != nil {
		return err
	}
	for i := range present {
		if err := renderFinding(w, &present[i]); err != nil {
			return err
		}
	}
	return nil
}

func renderFinding(w io.Writer, f *Finding) error {
	head := "  · " + f.Key + " (" + f.Metric + ")"
	if f.Unit != "" {
		head += " in " + f.Unit
	}
	if _, err := fmt.Fprintln(w, head); err != nil {
		return err
	}
	for _, r := range f.Readings {
		if _, err := fmt.Fprintf(w, "      %-14s %s\n", formatValue(r.Value), labelLine(r.Labels)); err != nil {
			return err
		}
	}
	if f.Note != "" {
		if _, err := fmt.Fprintf(w, "      note: %s\n", f.Note); err != nil {
			return err
		}
	}
	return nil
}

// renderMissing names what this deployment does not expose. Unavailable is never zero: a
// metric nobody published is not a metric measured at zero, and the difference decides whether
// a reader goes looking for a configuration flag or concludes their GPUs are idle.
func renderMissing(w io.Writer, s *Snapshot) error {
	missing := s.Missing()
	if len(missing) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(w, "\n  unavailable here -- not zero, not measured (%d):\n", len(missing)); err != nil {
		return err
	}
	for i := range missing {
		if _, err := fmt.Fprintf(w, "  · %-24s %s\n", missing[i].Key, missing[i].Metric); err != nil {
			return err
		}
	}
	return nil
}

// formatValue renders a reading without inventing precision: an integer prints as one, and a
// fraction keeps enough digits to be the number the exporter published.
func formatValue(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// labelLine renders a reading's identity in a stable order, so two runs of the same snapshot
// print the same lines.
func labelLine(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, " ")
}
