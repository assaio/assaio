package dashboard

import "strings"

// asciiDash is how every validator writes a parenthetical break in internal/analyze: the
// text report prints to terminals that cannot be assumed to render U+2014.
const asciiDash = " -- "

// prose renders validator text for the page. The dashboard is HTML, so the ASCII stand-in
// becomes the real em dash it always meant -- only when spaced, leaving `--flag` intact.
func prose(s string) string {
	return strings.ReplaceAll(s, asciiDash, " — ")
}
