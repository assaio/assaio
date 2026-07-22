package plugin

import (
	"strings"
	"testing"
)

func TestCappedBufferRejectsWritesPastCap(t *testing.T) {
	b := &cappedBuffer{cap: 10}
	if _, err := b.Write([]byte("12345")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte("678901")); err == nil {
		t.Fatal("Write past cap: err = nil, want errStdoutCapped")
	}
	if !b.exceeded {
		t.Fatal("exceeded = false, want true")
	}
	if b.buf.String() != "12345" {
		t.Fatalf("buf = %q, want the pre-cap content only", b.buf.String())
	}
}

func TestPrefixWriterPrefixesEveryLine(t *testing.T) {
	var out strings.Builder
	pw := newPrefixWriter(&out, "[plugin/demo] ")
	if _, err := pw.Write([]byte("one\ntwo\npar")); err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write([]byte("tial")); err != nil {
		t.Fatal(err)
	}
	pw.Flush()
	want := "[plugin/demo] one\n[plugin/demo] two\n[plugin/demo] partial\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

// TestPrefixWriterBoundsNewlineFreeFlood guards the stderr cap: a plugin writing an
// endless newline-free stream must not grow prefixWriter.partial without bound -- once it
// passes maxStderrLine it is force-flushed and the buffer reset.
func TestPrefixWriterBoundsNewlineFreeFlood(t *testing.T) {
	var out strings.Builder
	pw := newPrefixWriter(&out, "[p] ")
	big := strings.Repeat("A", maxStderrLine+1024)
	if _, err := pw.Write([]byte(big)); err != nil {
		t.Fatal(err)
	}
	if pw.partial.Len() > maxStderrLine {
		t.Fatalf("partial = %d bytes, want <= %d (flushed on overflow)", pw.partial.Len(), maxStderrLine)
	}
	if out.Len() == 0 {
		t.Fatal("flooded stderr must be force-flushed to the writer, not retained silently")
	}
}

func TestScanOutputStdoutWithinCapParses(t *testing.T) {
	out := []byte(`{"assaio_plugin":1,"tool":"demo"}` + "\n" +
		`{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","dedupe_key":"s1:0","granularity":"turn"}` + "\n")
	recs, _, stats, err := scanOutput(out, "demo", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || stats.Records != 1 {
		t.Fatalf("recs = %d stats = %+v, want 1 record", len(recs), stats)
	}
}
