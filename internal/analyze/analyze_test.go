package analyze

import "testing"

type fakeValidator struct{ name string }

func (f fakeValidator) Name() string     { return f.name }
func (f fakeValidator) Title() string    { return "Fake" }
func (f fakeValidator) Describe() string { return "test fixture, not a real metric" }
func (f fakeValidator) Analyze(Input) Result {
	return Result{Name: f.name, Title: "Fake", Takeaway: "fake"}
}

func TestRegisterAndGet(t *testing.T) {
	Register(fakeValidator{name: "zzz-test-fake"})
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
	Register(fakeValidator{name: "aaa-test-fake"})
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
	Register(fakeValidator{name: "copy-test-fake"})
	a := Validators()
	for i := range a {
		if a[i].Name() == "copy-test-fake" {
			a[i] = fakeValidator{name: "mutated"}
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
	v := fakeValidator{name: "result-shape-fake"}
	got := v.Analyze(Input{})
	if got.Name != "result-shape-fake" || got.Takeaway != "fake" {
		t.Fatalf("Analyze(Input{}) = %+v, want a Result carrying Name/Takeaway", got)
	}
}
