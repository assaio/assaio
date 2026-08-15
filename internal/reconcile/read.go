package reconcile

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// maxExportBytes bounds a file this package will read into memory. A billing export is a
// few thousand rows; anything past this is not one, and reading it would be the caller's
// problem to notice only after it had already happened. A var so a test can reach the bound
// without writing 64 MiB to disk; nothing outside this package can set it.
var maxExportBytes int64 = 64 << 20

// ReadFile reads a vendor export from path, choosing the reader by extension. overrides
// come from --map and win over every header alias.
func ReadFile(path string, overrides map[string]string) (*Export, error) {
	f, err := os.Open(path) //nolint:gosec // the export path is the operator's own downloaded file, named on the command line.
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	limited := &countingReader{r: io.LimitReader(f, maxExportBytes+1)}
	var exp *Export
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		exp, err = readJSON(limited, overrides)
	default:
		exp, err = readCSV(limited, overrides)
	}
	if err != nil {
		return nil, err
	}
	// The LimitReader's spare byte is what tells an oversized file from one ending exactly on the
	// bound: csv.Reader reads a cut as a clean io.EOF, so without this a truncated export
	// reconciles as a complete one.
	if limited.n > maxExportBytes {
		return nil, fmt.Errorf("export %s is larger than the %d MiB this command reads; split it or narrow the exported range",
			path, maxExportBytes>>20)
	}
	exp.Source = path
	return exp, nil
}

// countingReader records how much was actually read, which is what turns the LimitReader's
// spare byte into a detectable overrun rather than a silent truncation.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// readCSV parses a header row plus data rows. A row that cannot be parsed is skipped and
// counted, never guessed at -- the same policy every session-log parser follows.
func readCSV(r io.Reader, overrides map[string]string) (*Export, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	headers, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header row: %w", err)
	}
	binding, cols, err := bindColumns(headers, overrides)
	if err != nil {
		return nil, err
	}
	exp := &Export{Binding: binding}
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			exp.Skipped++
			exp.Total++
			continue
		}
		exp.Total++
		exp.add(func(field string) string { return at(rec, cols, field) })
	}
	return exp, nil
}

// readJSON parses an array of flat objects, taking the union of their keys as the header
// row so a field absent from the first object still binds.
func readJSON(r io.Reader, overrides map[string]string) (*Export, error) {
	var raw []map[string]any
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode export: %w", err)
	}
	seen := map[string]bool{}
	var headers []string
	for _, obj := range raw {
		for k := range obj {
			if !seen[k] {
				seen[k] = true
				headers = append(headers, k)
			}
		}
	}
	binding, _, err := bindColumns(headers, overrides)
	if err != nil {
		return nil, err
	}
	exp := &Export{Binding: binding}
	byField := map[string]string{
		FieldDay: binding.Day, FieldCost: binding.Cost, FieldModel: binding.Model,
		FieldTokens: binding.Tokens, FieldCurrency: binding.Currency,
	}
	for _, obj := range raw {
		exp.Total++
		exp.add(func(field string) string { return scalar(obj[byField[field]]) })
	}
	return exp, nil
}

// add builds one Row from a per-field accessor, skipping and counting anything whose date
// or money it cannot read.
func (e *Export) add(get func(field string) string) {
	d, ok := parseDay(get(FieldDay))
	if !ok {
		e.Skipped++
		return
	}
	cost, ok := parseMoney(get(FieldCost))
	if !ok {
		e.Skipped++
		return
	}
	row := Row{Day: d, Model: strings.TrimSpace(get(FieldModel)), Cost: cost}
	if n, stated := parseCount(get(FieldTokens)); stated {
		row.Tokens, row.TokensStated = n, true
	}
	if c := strings.ToUpper(strings.TrimSpace(get(FieldCurrency))); c != "" && e.Currency == "" {
		e.Currency = c
	}
	e.Rows = append(e.Rows, row)
}

func at(rec []string, cols map[string]int, field string) string {
	i, ok := cols[field]
	if !ok || i >= len(rec) {
		return ""
	}
	return rec[i]
}

// scalar renders a JSON value as text, leaving anything that is not a scalar empty rather
// than stringifying a structure into a field that expects a number.
func scalar(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return ""
	}
}
