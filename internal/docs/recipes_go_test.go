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

	// Read out of internal/analyze/analyze.go rather than written down here. A hand-copied
	// method set is how this stayed green while every published example failed to compile: the
	// Validator interface gained a method and the copy did not.
	want := validatorInterface(t)

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
	return typeSignature(src, fn.Type)
}

// typeSignature renders a func type as parameter and result types only. Parameter names are
// dropped because `Analyze(in Input) Result` and `Analyze(Input) Result` are the same signature,
// and an example is entitled to name what it takes.
func typeSignature(src string, fn *ast.FuncType) string {
	var params []string
	for _, f := range fn.Params.List {
		text := strings.TrimSpace(src[f.Type.Pos()-1 : f.Type.End()-1])
		for range max(len(f.Names), 1) {
			params = append(params, text)
		}
	}
	out := "(" + strings.Join(params, ", ") + ")"
	if fn.Results == nil {
		return out
	}
	var results []string
	for _, f := range fn.Results.List {
		text := strings.TrimSpace(src[f.Type.Pos()-1 : f.Type.End()-1])
		for range max(len(f.Names), 1) {
			results = append(results, text)
		}
	}
	if len(results) == 1 {
		return out + " " + results[0]
	}
	return out + " (" + strings.Join(results, ", ") + ")"
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

// validatorInterface is the Validator method set as analyze declares it in source, keyed by
// method name with the parameter list and result as written. Source text rather than reflection,
// because what a reader copies is text.
func validatorInterface(t *testing.T) map[string]string {
	t.Helper()
	src := read(t, repoRoot+"/internal/analyze/analyze.go")
	file, err := parser.ParseFile(token.NewFileSet(), "analyze.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || spec.Name.Name != "Validator" {
			return true
		}
		iface, ok := spec.Type.(*ast.InterfaceType)
		if !ok {
			return false
		}
		for _, m := range iface.Methods.List {
			fn, isFunc := m.Type.(*ast.FuncType)
			if !isFunc || len(m.Names) == 0 {
				continue
			}
			out[m.Names[0].Name] = typeSignature(src, fn)
		}
		return false
	})
	if len(out) == 0 {
		t.Fatal("no Validator interface found in internal/analyze/analyze.go")
	}
	return out
}
