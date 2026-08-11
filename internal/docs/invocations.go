package docs

import (
	"fmt"
	"regexp"
	"strings"
)

// Every command line a document prints is an instruction a reader will paste. A flag that was
// renamed, or a command that never existed, is a defect the reader discovers instead of the
// build -- which is the same failure the published surfaces had before they were checked. This
// reads the invocations out of the prose and holds each one to the command tree.
var invocation = regexp.MustCompile(`\bassaio(?:-agent)?\s+([a-z][^\n\x60|>]*)`)

// CheckInvocations reports every `assaio-agent …` line in a document that names a command or a
// flag the binary does not have.
func CheckInvocations(ref *Reference, file, content string) []Problem {
	ix := newIndex(ref)
	flags := flagIndex(ref)
	var problems []Problem
	for _, m := range invocation.FindAllStringSubmatch(content, -1) {
		path, args := splitInvocation(ix, strings.Fields(m[1]))
		if path == "" {
			continue // a prose sentence that happens to follow the binary's name
		}
		for _, arg := range args {
			name, ok := flagArg(arg)
			if !ok {
				continue
			}
			if flags[path][name] || flags[""][name] {
				continue
			}
			problems = append(problems, Problem{File: file, Text: fmt.Sprintf(
				"prints `assaio-agent %s %s`, and %s has no such flag", path, arg, path,
			)})
		}
	}
	return problems
}

// splitInvocation takes the longest run of leading words that names a real command, so
// "signals coverage --since 7d" is read as the subcommand it is rather than as `signals` with a
// stray word. Words after the command are arguments, which may be flags or values.
func splitInvocation(ix *index, words []string) (path string, args []string) {
	for i := range words {
		if strings.HasPrefix(words[i], "-") {
			break
		}
		candidate := strings.Join(words[:i+1], " ")
		if ix.ids["command."+candidate] {
			path, args = candidate, words[i+1:]
		}
	}
	return path, args
}

// flagArg reports the flag a word names, if it is one. A value written as --flag=value is the
// same flag, and a bare word is a value or a positional argument.
func flagArg(word string) (string, bool) {
	if !strings.HasPrefix(word, "--") || word == "--" {
		return "", false
	}
	name, _, _ := strings.Cut(strings.TrimPrefix(word, "--"), "=")
	name = strings.TrimRight(name, ".,;:)")
	if name == "" {
		return "", false
	}
	return name, true
}

// flagIndex maps a command path to its flags; the empty path holds the ones every command takes.
func flagIndex(ref *Reference) map[string]map[string]bool {
	out := map[string]map[string]bool{"": {}}
	for _, f := range ref.GlobalFlags {
		out[""][f.Name] = true
	}
	// --help is cobra's and reaches every command, but the reference leaves it out of each
	// command's own list, so a document printing it would otherwise read as wrong.
	out[""]["help"] = true
	for _, c := range ref.Commands {
		out[c.Path] = map[string]bool{}
		for _, f := range c.Flags {
			out[c.Path][f.Name] = true
		}
	}
	return out
}
