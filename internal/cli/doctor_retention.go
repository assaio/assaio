package cli

import (
	"strconv"
	"time"

	"github.com/assaio/assaio/internal/parser/claude"
)

// retentionLine states what limits how far back any report can ever go: not the install date, but
// how long the tool keeps its own transcripts. Printed whether or not anything has been lost,
// because "offline-first sees everything" is the belief this corrects, and a line that appears only
// after the loss cannot correct it in advance (B156).
//
// The comparison runs against claude-code's own oldest row rather than the whole store's: each
// source keeps its logs for its own length of time, and 45 days of store span on the audited
// machine came from three sources with three different answers.
//
// oldest is the earliest claude-code row; a zero time means nothing of its is stored.
func retentionLine(home string, oldest, now time.Time) string {
	r := claude.ReadRetention(home)
	line := "claude-code deletes its own transcripts after " + strconv.Itoa(r.Days) + " day(s)" + retentionOrigin(r)
	if oldest.IsZero() {
		return line + "; nothing of its is stored yet to compare against that."
	}
	held := int(now.Sub(oldest).Hours() / 24)
	line += "; this store holds " + strconv.Itoa(held) + " day(s) of it, from " + oldest.UTC().Format("2006-01-02") + "."
	if r.Days > 0 && held > r.Days {
		return line + "\n              " + strconv.Itoa(held-r.Days) + " of those days are older than the source still keeps," +
			" so they came from transcripts already deleted and exist only here now -- a `clear` or a lost store" +
			" ends them permanently. Nothing before " + oldest.UTC().Format("2006-01-02") + " can be recovered by any means."
	}
	return line + "\n              Older history is unrecoverable: the source deletes it, and nothing re-reads a deleted transcript."
}

func retentionOrigin(r claude.Retention) string {
	if r.Source == "" {
		return " (its default; no settings file sets cleanupPeriodDays)"
	}
	return " (cleanupPeriodDays in " + r.Source + ")"
}
