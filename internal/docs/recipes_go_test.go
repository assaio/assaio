package docs_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/docs"
)

// A Go example in a document cannot be compiled without a package around it, so this is the
// strongest check available: parse it, and hold its method set to the Validator interface as
// analyze declares it. That catches a renamed method and a changed signature -- the shape a
// reader's paste fails on. It does not catch a wrong number, which is what the honesty rules and
// a reviewer are for, and the page says exactly that rather than implying more.
func TestRecipeValidatorsMatchTheInterface(t *testing.T) {
	recipes := docs.ExtractRecipes(read(t, repoRoot+"/docs/recipes/extensions.md"))

	// The interface as internal/analyze/analyze.go declares it. Written out rather than
	// reflected over, because reflection sees the compiled signature and this has to compare
	// against source text a reader will copy.
	want := map[string]string{
		"Name":     "() string",
		"Title":    "() string",
		"Describe": "() string",
		"Analyze":  "(in Input) Result",
	}

	for _, name := range []string{"weekday-split", "answers-gated", "read-repeats"} {
		t.Run(name, func(t *testing.T) {
			block, err := recipes.Get(name)
			if err != nil {
				t.Fatal(err)
			}
			file, err := parser.ParseFile(token.NewFileSet(), name+".go", block, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("the published example does not parse: %v", err)
			}

			got := map[string]string{}
			registered := false
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if fn.Recv == nil {
					registered = registered || callsRegister(fn)
					continue
				}
				got[fn.Name.Name] = signature(block, fn)
			}

			for method, sig := range want {
				switch have, ok := got[method]; {
				case !ok:
					t.Errorf("the example declares no %s method; Validator requires %s%s", method, method, sig)
				case have != sig:
					t.Errorf("%s%s, and Validator requires %s%s", method, have, method, sig)
				}
			}
			if !registered {
				t.Error("the example never calls Register, so nothing would run it -- which is the " +
					"one line that makes a validator appear in analyze and the dashboard")
			}
		})
	}
}

// signature renders a method's parameters and result the way the document writes them, so the
// comparison is against the text a reader copies rather than against a resolved type.
func signature(src string, fn *ast.FuncDecl) string {
	return strings.TrimSpace(src[fn.Type.Params.Pos()-1 : fn.Type.End()-1])
}

func callsRegister(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "Register" {
			found = true
		}
		return true
	})
	return found
}
