// Package parser holds helpers shared by the per-tool log parsers.
package parser

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
	"math"
)

// MaxVocabularyToken caps a vendor category label. Real ones are a couple of words; the cap
// exists because the line carrying one may be up to MaxLineBytes.
const MaxVocabularyToken = 64

// MaxLineBytes caps a single log line; a longer line aborts the file scan.
const MaxLineBytes = 16 * 1024 * 1024

// NewScanner returns a line scanner over r, buffered up to MaxLineBytes.
func NewScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 1024*1024), MaxLineBytes)
	return sc
}

// NonNeg clamps n to zero, dropping negative deltas rather than propagating them.
func NonNeg(n int64) int64 {
	if n < 0 {
		return 0
	}
	return n
}

// SumNonNeg adds already-clamped counts, saturating at MaxInt64 instead of wrapping. Summing
// what a record keeps as separate columns is what makes this necessary: four fields a log can
// each state as MaxInt64 overflow into a negative total, which a fuzzer found within a second
// of the step timeline first summing them.
func SumNonNeg(ns ...int64) int64 {
	var total int64
	for _, n := range ns {
		n = NonNeg(n)
		if total > math.MaxInt64-n {
			return math.MaxInt64
		}
		total += n
	}
	return total
}

// VocabularyToken keeps a vendor's category label only when it has the shape of one --
// lowercase, digits and underscores, bounded. A label is rendered verbatim and grouped on, so
// a log carrying free text (or a crafted line) in a field documented as a closed vocabulary
// must not reach the store; the honest reading of an unrecognisable shape is that this turn
// stated no label, which every figure over it already renders as an absence.
func VocabularyToken(s string) string {
	if len(s) > MaxVocabularyToken {
		return ""
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return ""
		}
	}
	return s
}

// Subset clamps a count the log reports as a portion of total to that total: whatever is
// left is read as the remainder, and a portion larger than its whole makes that negative.
func Subset(n, total int64) int64 {
	if n > total {
		return NonNeg(total)
	}
	return NonNeg(n)
}

// FileFingerprint returns a short deterministic digest of b. Parsers that build a
// DedupeKey from a positional counter (e.g. "session:index") prefix it with the
// fingerprint of the first line they read, so the same session/task id reused across two
// different files (e.g. a resumed session logged to a new file) never collides.
func FileFingerprint(b []byte) string {
	h := fnv.New32a()
	_, _ = h.Write(b)
	return fmt.Sprintf("%08x", h.Sum32())
}
