package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// TestAnUndeclaredValidatorIsNeverWithheld is the property that makes this migratable: a
// validator that has not declared what it reads behaves exactly as it did before, on every
// window. "Needs everything" as the default would withhold most of the catalogue the moment a
// store had no sub-agent attribution.
func TestAnUndeclaredValidatorIsNeverWithheld(t *testing.T) {
	in := BuildInput(nil, nil, pricing.Table{}, time.Now(), 7*24*time.Hour, Delegation{})
	for _, v := range Validators() {
		if _, declares := v.(Needs); declares {
			continue
		}
		if got := Missing(v, &in); len(got) != 0 {
			t.Fatalf("%s declared nothing but Missing reported %v", v.Name(), got)
		}
	}
}

// TestADeclaredValidatorSaysWhatTheWindowCouldNotSupply is the reason to declare: a metric that
// measured its subject and found nothing, and one that was never handed the evidence, are
// different facts a reader acts on differently.
func TestADeclaredValidatorSaysWhatTheWindowCouldNotSupply(t *testing.T) {
	in := BuildInput(nil, nil, pricing.Table{}, time.Now(), 7*24*time.Hour, Delegation{})
	v := mustGet(t, skillName)
	got := Evaluate(v, &in)

	if len(got.Withheld) != 1 || got.Withheld[0] != CapAttribution {
		t.Fatalf("Withheld = %v, want [%s]", got.Withheld, CapAttribution)
	}
	if !strings.Contains(strings.Join(got.Caveats, " "), "Withheld input") {
		t.Fatalf("Caveats = %q, want the withheld-input disclosure", got.Caveats)
	}
}

// TestNothingIsWithheldWhenTheWindowCanAnswer keeps the disclosure from becoming noise every
// reader learns to skip.
func TestNothingIsWithheldWhenTheWindowCanAnswer(t *testing.T) {
	in := BuildInput(nil, nil, pricing.Table{}, time.Now(), 7*24*time.Hour, Delegation{})
	in.Skills = []store.AttributionRow{{Name: "review", Tokens: 500_000, Sessions: 4}}
	in.Agents = []store.AttributionRow{{Name: "explorer", Tokens: 400_000, Sessions: 3}}
	got := Evaluate(mustGet(t, skillName), &in)

	if len(got.Withheld) != 0 {
		t.Fatalf("Withheld = %v on a window that carries attribution", got.Withheld)
	}
}

// TestEveryDeclaredCapabilityIsInTheClosedSet stops a validator inventing a capability name no
// registry publishes and no plugin could ever declare back.
func TestEveryDeclaredCapabilityIsInTheClosedSet(t *testing.T) {
	for _, v := range Validators() {
		for _, c := range NeededBy(v) {
			if !ValidCapability(c) {
				t.Errorf("%s declares %q, which is not one of %v", v.Name(), c, Capabilities())
			}
		}
	}
}
