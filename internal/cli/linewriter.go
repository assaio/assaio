package cli

import (
	"fmt"
	"io"
)

// lineWriter collects sequential write errors so a renderer can emit many lines and
// report only the first failure, keeping the io.Writer error contract without a check
// after every line (Rob Pike's errWriter pattern).
type lineWriter struct {
	w   io.Writer
	err error
}

func (lw *lineWriter) printf(format string, a ...any) {
	if lw.err != nil {
		return
	}
	_, lw.err = fmt.Fprintf(lw.w, format, a...)
}

func (lw *lineWriter) println(s string) {
	if lw.err != nil {
		return
	}
	_, lw.err = fmt.Fprintln(lw.w, s)
}
