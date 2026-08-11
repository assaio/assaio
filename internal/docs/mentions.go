package docs

import (
	"fmt"
	"regexp"
	"strings"
)

// A file that cannot carry attributes -- plain text, Markdown -- gets the weaker half of the
// same rule: presence, not position. A document that never says the words has not heard of the
// thing, which is the state `digest` and `mark --suggest` left the published surfaces in for a
// whole release. Two matchers, because a set is written either as English or as an identifier,
// and one matcher for both would either demand prose spell "claude-code" or accept the word
// "skills" as evidence that a wire field is documented.

// CheckMentions requires every member of each set to be named the way prose names it:
// case-folded, with separators treated alike, so "Claude Code" satisfies "claude-code".
func CheckMentions(ref *Reference, file, content string, sets ...string) []Problem {
	return check(ref, file, sets, func(name string) bool {
		return strings.Contains(pad(normalize(content)), pad(normalize(name)))
	})
}

// CheckIdentifiers requires every member of each set to appear inside a code span or a fenced
// block. The English word "skills" in a sentence is not evidence that the `skills` field is
// documented, and a matcher that accepted it would be green on a document that never mentioned
// the field at all. Anywhere inside the code region counts, so `barsAreProjects: true` and a
// JSON example both do -- demanding the bare token would make the check a house style rather
// than a question about coverage.
func CheckIdentifiers(ref *Reference, file, content string, sets ...string) []Problem {
	code := codeRegions(content)
	return check(ref, file, sets, func(name string) bool {
		return regexp.MustCompile(`(^|\W)` + regexp.QuoteMeta(name) + `($|\W)`).MatchString(code)
	})
}

var (
	fencedBlock = regexp.MustCompile("(?s)```.*?```")
	inlineSpan  = regexp.MustCompile("`[^`\n]+`")
)

// codeRegions is every fenced block and inline code span in a Markdown document, joined. What
// falls outside it is prose, and prose naming a field is not the field being documented.
func codeRegions(content string) string {
	var b strings.Builder
	for _, block := range fencedBlock.FindAllString(content, -1) {
		b.WriteString(block)
		b.WriteByte('\n')
	}
	for _, span := range inlineSpan.FindAllString(fencedBlock.ReplaceAllString(content, ""), -1) {
		b.WriteString(span)
		b.WriteByte('\n')
	}
	return b.String()
}

func check(ref *Reference, file string, sets []string, found func(name string) bool) []Problem {
	ix := newIndex(ref)
	var problems []Problem
	for _, set := range sets {
		members, ok := ix.sets[set]
		if !ok {
			problems = append(problems, Problem{
				File: file,
				Text: fmt.Sprintf("is checked against %q, which is not a set. %s", set, ix.setNames()),
			})
			continue
		}
		for _, m := range members {
			if exempt[m.id] != "" || found(m.name) {
				continue
			}
			problems = append(problems, Problem{File: file, Text: fmt.Sprintf(
				"never mentions %q from %q -- say what it is, or exempt it in internal/docs with the reason",
				m.name, set,
			)})
		}
	}
	return problems
}

var separators = regexp.MustCompile(`[^a-z0-9]+`)

func normalize(s string) string {
	return strings.TrimSpace(separators.ReplaceAllString(strings.ToLower(s), " "))
}

// pad makes a normalized substring match respect word boundaries, so "cline" is not found
// inside "declines".
func pad(s string) string { return " " + s + " " }
