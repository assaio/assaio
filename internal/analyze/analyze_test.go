package analyze

import (
	"fmt"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/layer"
)

type fakeValidator struct {
	name string
	// declares is stated at every call site rather than defaulted, so a fixture built to
	// exercise the layer check cannot be mistaken for one that forgot to set it.
	declares layer.Layer
}

func (f fakeValidator) Name() string       { return f.name }
func (f fakeValidator) Title() string      { return "Fake" }
func (f fakeValidator) Describe() string   { return "test fixture, not a real metric" }
func (f fakeValidator) Layer() layer.Layer { return f.declares }
func (f fakeValidator) Analyze(Input) Result {
	return Result{Name: f.name, Title: "Fake", Takeaway: "fake"}
}

// registerForTest registers v and puts the registry back when the test ends. The registry is
// package state every other test in this package reads through Validators(), and a fixture left
// behind in it gets asked to answer windows it was never written for.
func registerForTest(t *testing.T, v Validator) {
	t.Helper()
	saved := registry
	t.Cleanup(func() { registry = saved })
	Register(v)
}

func TestRegisterAndGet(t *testing.T) {
	registerForTest(t, fakeValidator{name: "zzz-test-fake", declares: layer.Activity})
	v, ok := Get("zzz-test-fake")
	if !ok {
		t.Fatal("Get must find a just-registered validator")
	}
	if v.Name() != "zzz-test-fake" {
		t.Fatalf("Name() = %q, want zzz-test-fake", v.Name())
	}
}

func TestGetUnknownNameReportsFalse(t *testing.T) {
	if _, ok := Get("no-such-validator-xyz"); ok {
		t.Fatal("Get must report false for an unregistered name")
	}
}

func TestValidatorsSortedByName(t *testing.T) {
	registerForTest(t, fakeValidator{name: "aaa-test-fake", declares: layer.Activity})
	all := Validators()
	for i := 1; i < len(all); i++ {
		if all[i-1].Name() > all[i].Name() {
			t.Fatalf("Validators() not sorted by Name: %q before %q", all[i-1].Name(), all[i].Name())
		}
	}
}

// TestValidatorsReturnsStableCopy asserts mutating the returned slice never corrupts the
// live registry -- Validators() must hand back a copy, not internal state.
func TestValidatorsReturnsStableCopy(t *testing.T) {
	registerForTest(t, fakeValidator{name: "copy-test-fake", declares: layer.Activity})
	a := Validators()
	for i := range a {
		if a[i].Name() == "copy-test-fake" {
			a[i] = fakeValidator{name: "mutated", declares: layer.Activity}
		}
	}
	b := Validators()
	if _, ok := Get("copy-test-fake"); !ok {
		t.Fatal("mutating Validators()'s result must not affect the registry Get reads from")
	}
	for _, v := range b {
		if v.Name() == "mutated" {
			t.Fatal("Validators() must return a copy, not the live registry slice")
		}
	}
}

func TestBuiltinValidatorsRegistered(t *testing.T) {
	for _, name := range []string{"adoption", "model-fit", "context", "throughput", "rework"} {
		v, ok := Get(name)
		if !ok {
			t.Fatalf("built-in validator %q must be registered", name)
		}
		if v.Title() == "" || v.Describe() == "" {
			t.Fatalf("built-in validator %q must have a non-empty Title and Describe", name)
		}
	}
}

// TestFakeValidatorAnalyzeReturnsResult asserts the Validator interface's Analyze method
// returns a plain Result value (no Report interface / RenderText indirection).
func TestFakeValidatorAnalyzeReturnsResult(t *testing.T) {
	v := fakeValidator{name: "result-shape-fake", declares: layer.Activity}
	got := v.Analyze(Input{})
	if got.Name != "result-shape-fake" || got.Takeaway != "fake" {
		t.Fatalf("Analyze(Input{}) = %+v, want a Result carrying Name/Takeaway", got)
	}
}

// Register's two start-time invariants, both of them panics: neither a layer outside the closed
// vocabulary nor a name a validator already holds can be reported around at run time. A second
// validator under a taken name is the one that looks harmless -- both run, Get answers for the
// first, and the document carries the key twice with nothing saying which verdict is which.
func TestRegisterRejectsAnInvalidDeclaration(t *testing.T) {
	tests := []struct {
		name string
		v    fakeValidator
		// want is a substring the panic message must carry; "" means Register must accept v.
		want string
	}{
		{"a free name on a declared layer", fakeValidator{name: "register-fresh-fake", declares: layer.Output}, ""},
		{"a fifth layer nobody defined", fakeValidator{name: "register-layer-fake", declares: "velocity"}, "not one of"},
		{"no layer at all", fakeValidator{name: "register-nolayer-fake", declares: ""}, "not one of"},
		// A different concrete type under a built-in's name: the collision is on Name(), which
		// is what a fork adding a metric beside the built-ins actually hits.
		{"a name a built-in already holds", fakeValidator{name: "adoption", declares: layer.Activity}, "already registered"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := registry
			t.Cleanup(func() { registry = saved })

			got := panicMessage(func() { Register(tt.v) })
			if tt.want == "" {
				if got != "" {
					t.Fatalf("Register panicked on a valid declaration: %s", got)
				}
				if _, ok := Get(tt.v.name); !ok {
					t.Fatalf("Register accepted %q but Get cannot find it", tt.v.name)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Fatalf("panic = %q, want it to carry %q", got, tt.want)
			}
			if !strings.Contains(got, tt.v.name) {
				t.Errorf("panic = %q, want it to name the offending validator %q", got, tt.v.name)
			}
			if len(registry) != len(saved) {
				t.Errorf("a rejected validator still reached the registry: %d entries, want %d",
					len(registry), len(saved))
			}
		})
	}
}

// panicMessage returns what f panicked with, or "" when it returned normally.
func panicMessage(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
		}
	}()
	f()
	return ""
}
