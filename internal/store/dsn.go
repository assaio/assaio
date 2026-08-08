package store

import "strings"

// storePragmas are the connection settings every assaio store runs with, spelled once so a
// caller cannot open a database without them.
const storePragmas = "_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"

// pathEscaper escapes the three characters that change what a SQLite URI means. Everything
// after "file:" is parsed as a URI, so an unescaped '?' starts the query and '#' starts the
// fragment: either truncates the filename and opens a database somewhere else while still
// reporting success, and '?' takes the pragmas with it. '%' goes first because the URI is
// percent-decoded after that split, so a literal one would otherwise introduce an escape
// the path never contained.
var pathEscaper = strings.NewReplacer("%", "%25", "?", "%3F", "#", "%23")

// storeDSN builds the connection string for a database file at path. Spaces, backslashes
// and every other character are left alone: SQLite accepts them, and rewriting them would
// change which file a Windows or a space-containing path resolves to.
func storeDSN(path string) string {
	return "file:" + pathEscaper.Replace(path) + "?" + storePragmas
}
