package docs

import (
	"regexp"
	"strings"
)

// The checker reads markup with a scanner rather than a parser, which is enough for the one
// question it asks -- but only if two things hold. A commented-out claim must not count, and a
// claim must count for the container that actually encloses it: "this list enumerates them" is
// the promise, and a claim two sections away does not keep it. Both were true of the first
// version of this file and are what these two functions exist to make false.

var (
	claimAttr   = regexp.MustCompile(`data-claim="([^"]*)"`)
	coversAttr  = regexp.MustCompile(`data-covers="([^"]*)"`)
	htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	openTagName = regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9]*)`)
)

// stripComments blanks out HTML comments, keeping the byte length so every offset below still
// points where it did. A claim inside a comment is invisible to a reader, so it may not satisfy
// a promise made to one.
func stripComments(content string) string {
	return htmlComment.ReplaceAllStringFunc(content, func(m string) string {
		return strings.Repeat(" ", len(m))
	})
}

// enclosure returns the substring of the element whose opening tag contains the attribute at
// `at`. It matches the element's own nested tags of the same name, so an outer list is not
// closed by an inner one. A malformed fragment with no closing tag yields the rest of the
// document, which fails open into the wider check rather than silently covering nothing.
func enclosure(content string, at int) string {
	start := strings.LastIndex(content[:at], "<")
	if start < 0 {
		return content[at:]
	}
	m := openTagName.FindStringSubmatch(content[start:])
	if m == nil {
		return content[at:]
	}
	name := m[1]
	body := strings.Index(content[start:], ">")
	if body < 0 {
		return content[at:]
	}
	rest := content[start+body+1:]

	open, close := "<"+name, "</"+name
	depth, i := 1, 0
	for depth > 0 {
		nextOpen := indexFrom(rest, open, i)
		nextClose := indexFrom(rest, close, i)
		if nextClose < 0 {
			return rest
		}
		if nextOpen >= 0 && nextOpen < nextClose {
			depth++
			i = nextOpen + len(open)
			continue
		}
		depth--
		i = nextClose + len(close)
	}
	return rest[:i-len(close)]
}

// indexFrom finds a tag prefix that is followed by a tag boundary, so <div> is not found inside
// <divider>.
func indexFrom(s, tag string, from int) int {
	for at := from; at < len(s); {
		i := strings.Index(s[at:], tag)
		if i < 0 {
			return -1
		}
		i += at
		next := i + len(tag)
		if next >= len(s) || s[next] == '>' || s[next] == ' ' || s[next] == '\n' || s[next] == '\t' {
			return i
		}
		at = next
	}
	return -1
}
