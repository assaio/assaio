package docs

import (
	"fmt"
	"regexp"
	"strings"
)

// A recipe in the cookbook is only worth reading if it still works. Each executable one is
// marked in its fence -- ```yaml #ticket-keys -- and a test runs that exact block, so the
// snippet a reader copies is the snippet the suite exercised. The marker is invisible on the
// page: a renderer takes only the first word of an info string as the language.
var recipeFence = regexp.MustCompile("(?m)^```[a-z]* #([a-z0-9-]+)[ \t]*\n")

// Recipe is one marked block, under the name its fence gives it.
type Recipe struct {
	Name string
	Body string
}

// Recipes is the marked set of one document.
type Recipes []Recipe

// ExtractRecipes reads every marked block from a document, in order.
func ExtractRecipes(doc string) Recipes {
	var out Recipes
	for _, loc := range recipeFence.FindAllStringSubmatchIndex(doc, -1) {
		body := doc[loc[1]:]
		end := strings.Index(body, "\n```")
		if end < 0 {
			continue
		}
		out = append(out, Recipe{Name: doc[loc[2]:loc[3]], Body: body[:end+1]})
	}
	return out
}

// Get returns one block by name, so a test names what it exercises rather than depending on
// the order the document happens to be written in.
func (r Recipes) Get(name string) (string, error) {
	for _, x := range r {
		if x.Name == name {
			return x.Body, nil
		}
	}
	return "", fmt.Errorf("no recipe named %q; this document has %s", name, r.Names())
}

// Names lists what the document offers, for an error a reader can act on.
func (r Recipes) Names() string {
	names := make([]string, 0, len(r))
	for _, x := range r {
		names = append(names, x.Name)
	}
	return strings.Join(names, ", ")
}
