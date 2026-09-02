package agy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/parser"
)

// sentinel is the string every content-bearing field of both fixtures is filled with -- the
// prompt, the model's thinking, and every tool-call argument. The real transcripts those
// fixtures were redacted from carry a person's prompt and the code the model wrote in exactly
// those positions.
const sentinel = "assaio-sentinel-prompt-body-must-not-escape"

// TestNoPromptContentEscapes is the first test to read in this package. Antigravity CLI is the
// first source whose accounting sits inside the same lines as the content, so the omission has
// to be structural rather than a review note: the decoder declares only the fields it needs, and
// content therefore has no path into a record, a returned error, or anything this package writes.
//
// The assertion is over everything the parser hands back, not over the fields a reader expects
// to be risky: a record is serialized whole, so a field added to usage.Record and populated from
// a content-bearing key fails here rather than shipping.
func TestNoPromptContentEscapes(t *testing.T) {
	for _, fixture := range []string{"testdata/session.jsonl", "testdata/interrupted.jsonl"} {
		t.Run(fixture, func(t *testing.T) {
			raw, err := os.ReadFile(fixture) //nolint:gosec // a fixture path this file lists
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(raw, []byte(sentinel)) {
				t.Fatalf("%s carries no sentinel, so this test proves nothing", fixture)
			}
			recs, _, parseErr := ParseTranscript(bytes.NewReader(raw), "conv-1")
			if len(recs) == 0 {
				t.Fatal("no records parsed, so this test proves nothing")
			}
			out, err := json.Marshal(recs)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(out, []byte(sentinel)) {
				t.Errorf("a parsed record carries content from the transcript: %s", out)
			}
			if parseErr != nil && strings.Contains(parseErr.Error(), sentinel) {
				t.Errorf("the returned error quotes content from the transcript: %v", parseErr)
			}
		})
	}
}

// TestScannerErrorNamesNoContent covers the one path that formats a message rather than a
// record. A parser that wraps a failing line quotes it, and the failing line here is a prompt:
// the scanner's error is about the file, so the wrap may name the file's shape and never its
// bytes.
func TestScannerErrorNamesNoContent(t *testing.T) {
	long := io.LimitReader(repeat(sentinel), parser.MaxLineBytes+1)
	_, _, err := ParseTranscript(long, "conv-1")
	if err == nil {
		t.Fatal("a line past the scanner cap must fail the file")
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Errorf("the scanner error quotes the line that overran: %v", err)
	}
}

// TestOpenErrorNamesNoContent: ParseDir builds its path from a directory name, so a missing or
// unreadable transcript must report the failure without the path itself becoming the message --
// a conversation directory is named for the conversation, not for its content, but the file
// path is still local filesystem detail that PRIVACY.md keeps out of stored and printed output.
func TestOpenErrorNamesNoContent(t *testing.T) {
	dir := t.TempDir()
	_, _, err := ParseDir(dir)
	if err == nil {
		t.Fatal("a directory with no transcript must fail")
	}
	if !strings.Contains(err.Error(), "agy transcript") {
		t.Errorf("error = %v, want it to name what could not be opened", err)
	}
	// The assertion the constraint actually needs: os.Open's own error is a *fs.PathError
	// whose message is the full path, so wrapping it put a person's home directory and the
	// conversation uuid into the message. Nothing prints it today; this is what keeps that
	// true when something does.
	if strings.Contains(err.Error(), dir) {
		t.Errorf("error = %v, want it to name the failure without the directory it happened in", err)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %v, want the reason still readable by errors.Is", err)
	}
}

// TestDecoderReadsOnlyTheAllowlistedKeys pins the boundary itself rather than its effect: the
// struct the transcript is decoded into is the allowlist, so a reviewer can check the list
// against the vendor's own keys in one place. content, thinking and every tool-call argument
// are absent from it by construction.
func TestDecoderReadsOnlyTheAllowlistedKeys(t *testing.T) {
	var l line
	got := fmt.Sprintf("%+v", l)
	for _, forbidden := range []string{"Content", "Thinking", "Args", "Payload"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("the decoded line carries a %s field, which is content", forbidden)
		}
	}
}

// repeat is an endless reader of s, so a line past the scanner cap costs no allocation to
// build.
type repeat string

func (r repeat) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r[i%len(r)]
	}
	return len(p), nil
}
