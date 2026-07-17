package plugin

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// maxStdout caps a plugin's stdout; a plugin that exceeds it is aborted with a clear
// error rather than let assaio buffer unbounded memory for a misbehaving subprocess.
const maxStdout = 64 * 1024 * 1024

// errStdoutCapped aborts the exec copy loop once a plugin exceeds maxStdout.
var errStdoutCapped = errors.New("plugin stdout cap exceeded")

// cappedBuffer buffers a plugin's stdout up to cap bytes, then rejects further writes
// so the subprocess is cut off instead of filling memory.
type cappedBuffer struct {
	buf      bytes.Buffer
	cap      int
	exceeded bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.cap {
		b.exceeded = true
		return 0, errStdoutCapped
	}
	return b.buf.Write(p)
}

// prefixWriter forwards writes to w line by line, each line prefixed, so a plugin's
// stderr stays attributable inside assaio's own stderr stream.
type prefixWriter struct {
	w       io.Writer
	prefix  string
	partial bytes.Buffer
}

func newPrefixWriter(w io.Writer, prefix string) *prefixWriter {
	return &prefixWriter{w: w, prefix: prefix}
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	pw.partial.Write(p)
	for {
		line, err := pw.partial.ReadString('\n')
		if err != nil {
			pw.partial.WriteString(line)
			break
		}
		_, _ = fmt.Fprintf(pw.w, "%s%s", pw.prefix, line)
	}
	return len(p), nil
}

// Flush emits any trailing line the plugin left without a newline.
func (pw *prefixWriter) Flush() {
	if pw.partial.Len() > 0 {
		_, _ = fmt.Fprintf(pw.w, "%s%s\n", pw.prefix, pw.partial.String())
		pw.partial.Reset()
	}
}
