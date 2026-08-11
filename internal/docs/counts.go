package docs

import (
	"regexp"
	"strconv"
	"strings"
)

// A count claim is checked against what the annotated element actually renders, because the
// sentence is what a reader believes -- "Nineteen reads" is a claim about the binary whether it
// is spelled or numbered. The element carrying a count claim must contain text and no nested
// markup; nothing else needs that, and requiring it keeps this a substring match rather than an
// HTML parser.
//
// Every occurrence is returned, not the first: a page states the same count in more than one
// sentence, and checking only one of them would let the others fall behind silently.
func elementTexts(content, claim string) []string {
	var out []string
	needle := `data-claim="` + claim + `"`
	for at := 0; ; {
		i := strings.Index(content[at:], needle)
		if i < 0 {
			return out
		}
		rest := content[at+i+len(needle):]
		open := strings.Index(rest, ">")
		at += i + len(needle)
		if open < 0 {
			return out
		}
		text := rest[open+1:]
		if end := strings.Index(text, "<"); end >= 0 {
			text = text[:end]
		}
		out = append(out, strings.TrimSpace(text))
	}
}

// mentionsCount reports whether text states n, as a numeral or as the English word a page is
// more likely to write it in.
func mentionsCount(text string, n int) bool {
	if hasWord(text, strconv.Itoa(n)) {
		return true
	}
	word := numberWord(n)
	return word != "" && hasWord(strings.ToLower(text), word)
}

func hasWord(text, want string) bool {
	return regexp.MustCompile(`(^|[^\w-])` + regexp.QuoteMeta(want) + `($|[^\w-])`).MatchString(text)
}

var (
	ones = []string{
		"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten",
		"eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen",
		"eighteen", "nineteen",
	}
	tens = []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}
)

// numberWord spells n in English, for 0..99. Beyond that a page writes a numeral, and an
// empty string means the numeral is the only accepted form.
func numberWord(n int) string {
	switch {
	case n < 0 || n > 99:
		return ""
	case n < len(ones):
		return ones[n]
	case n%10 == 0:
		return tens[n/10]
	default:
		return tens[n/10] + "-" + ones[n%10]
	}
}
