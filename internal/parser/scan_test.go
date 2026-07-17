package parser

import (
	"strings"
	"testing"
)

func TestNewScannerReadsLines(t *testing.T) {
	sc := NewScanner(strings.NewReader("a\nb\nc\n"))
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("got %d lines want 3: %v", len(lines), lines)
	}
}

func TestNonNeg(t *testing.T) {
	cases := map[int64]int64{-5: 0, 0: 0, 5: 5}
	for in, want := range cases {
		if got := NonNeg(in); got != want {
			t.Fatalf("NonNeg(%d) = %d want %d", in, got, want)
		}
	}
}

func TestFileFingerprintDeterministic(t *testing.T) {
	a := FileFingerprint([]byte(`{"id":"s1"}`))
	b := FileFingerprint([]byte(`{"id":"s1"}`))
	if a != b {
		t.Fatalf("FileFingerprint not deterministic: %q vs %q", a, b)
	}
}

func TestFileFingerprintDiffersByContent(t *testing.T) {
	a := FileFingerprint([]byte(`{"id":"s1","timestamp":"2026-07-01T09:00:00Z"}`))
	b := FileFingerprint([]byte(`{"id":"s1","timestamp":"2026-07-02T09:00:00Z"}`))
	if a == b {
		t.Fatalf("FileFingerprint collided for different input: %q", a)
	}
}
