package openmetrics

import "strings"

// The line-level grammar: how one exposition line is cut into a name, a label set and a value.
// It sits apart from the document-level parse because it is where every boundary check lives --
// a crafted metric name, a label value carrying a comma or a quote, a value that is not a number
// -- and those checks are the whole reason this package can be pointed at a network endpoint.

// splitSample separates the metric name, its labels and the remainder of the line.
func splitSample(line string, lim Limits) (name string, labels map[string]string, rest string, ok bool) {
	open := strings.IndexByte(line, '{')
	if open < 0 {
		fields := strings.SplitN(line, " ", 2)
		if len(fields) != 2 || !validName(fields[0]) {
			return "", nil, "", false
		}
		return fields[0], nil, fields[1], true
	}
	closeAt := strings.LastIndexByte(line, '}')
	if closeAt < open {
		return "", nil, "", false
	}
	name = strings.TrimSpace(line[:open])
	if !validName(name) {
		return "", nil, "", false
	}
	labels, ok = parseLabels(line[open+1:closeAt], lim)
	if !ok {
		return "", nil, "", false
	}
	return name, labels, strings.TrimSpace(line[closeAt+1:]), true
}

// parseLabels reads `a="1",b="2"`. A label set past the cardinality budget fails the sample
// rather than being trimmed: a half-identified sample is worse than a missing one, because
// nothing downstream could tell it apart from a fully identified one.
func parseLabels(s string, lim Limits) (map[string]string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, true
	}
	labels := map[string]string{}
	for _, pair := range splitLabelPairs(s) {
		eq := strings.IndexByte(pair, '=')
		if eq <= 0 {
			return nil, false
		}
		key := strings.TrimSpace(pair[:eq])
		value := strings.TrimSpace(pair[eq+1:])
		if !validName(key) || len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
			return nil, false
		}
		unquoted := value[1 : len(value)-1]
		if len(unquoted) > lim.MaxLabelValueBytes {
			return nil, false
		}
		labels[key] = unescape(unquoted)
		if len(labels) > lim.MaxLabelsPerSample {
			return nil, false
		}
	}
	return labels, true
}

// splitLabelPairs splits on commas that are outside a quoted value, so a label value holding a
// comma does not split the pair it belongs to.
func splitLabelPairs(s string) []string {
	var out []string
	var start int
	inQuotes, escaped := false, false
	for i := range len(s) {
		switch {
		case escaped:
			escaped = false
		case s[i] == '\\':
			escaped = true
		case s[i] == '"':
			inQuotes = !inQuotes
		case s[i] == ',' && !inQuotes:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// unescape resolves the three escapes the exposition format defines inside a label value.
func unescape(s string) string {
	r := strings.NewReplacer(`\\`, `\`, `\"`, `"`, `\n`, "\n")
	return r.Replace(s)
}

// validName holds a metric or label name to the character set the format defines, bounded.
// A crafted name reaches a rendered report, so this is a boundary check rather than a
// convenience.
func validName(s string) bool {
	if s == "" || len(s) > maxNameBytes {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_', r == ':':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

// maxNameBytes bounds a metric or label name. Real ones are a few dozen characters.
const maxNameBytes = 256
