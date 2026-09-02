package trace

import (
	"strconv"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/store"
)

// A window a scope covers whole must say so plainly, with no exclusion percentage for a reader
// to discount the figure by.
func TestCaveatOnAWindowTheScopeCoversWhole(t *testing.T) {
	set := New([]store.Timeline{seq("s1", "", entrypointCLI, origin, 4)})
	v := set.Scope(Interactive)

	got := v.Caveat()
	if !strings.Contains(got, "every sequence in this window") {
		t.Errorf("Caveat() = %q, want it to state the scope left nothing out", got)
	}
	if strings.Contains(got, "%") {
		t.Errorf("Caveat() = %q, want no exclusion share when nothing was excluded", got)
	}
	if !strings.Contains(got, scopeCovers[Interactive]) {
		t.Errorf("Caveat() = %q, want the scope in a reader's terms", got)
	}
}

// The excluded case is why every detector renders this rather than its own wording. The share
// quoted is the step share: it and the sequence share differ by an order of magnitude, and a
// detector free to phrase its own exclusion would sooner or later quote whichever flattered the
// finding.
func TestCaveatQuotesTheStepShareNotTheSequenceShare(t *testing.T) {
	sequences := []store.Timeline{seq("person", "", entrypointCLI, origin, 200)}
	for i := range 20 {
		sequences = append(sequences, seq("script"+strconv.Itoa(i), "", entrypointSDKPy, origin, 1))
	}
	set := New(sequences)
	v := set.Scope(Interactive)

	got := v.Caveat()
	// 200 of 220 steps are in scope (91%) while only 1 of 21 sequences is (5%, or 95% excluded).
	// Quoting the coverage in sequence terms here would understate the figure's reach eighteenfold.
	if !strings.Contains(got, "91%") {
		t.Errorf("Caveat() = %q, want the 91%% step coverage", got)
	}
	for _, sequenceShare := range []string{"5%", "95%"} {
		if strings.Contains(got, sequenceShare) {
			t.Errorf("Caveat() = %q quotes the sequence share %s; the two shares are not interchangeable",
				got, sequenceShare)
		}
	}
	if !strings.Contains(got, "20 sequence(s) are excluded") {
		t.Errorf("Caveat() = %q, want the excluded sequence count stated alongside", got)
	}
}

// A caveat is the one line standing between a figure and a reader, so an unnamed scope has to
// degrade to the raw name rather than to a sentence with a hole where the population goes.
func TestCaveatNamesAnUnknownScopeRatherThanLeavingABlank(t *testing.T) {
	set := New([]store.Timeline{seq("s1", "", entrypointCLI, origin, 4)})
	v := set.Scope("intractive")

	got := v.Caveat()
	if !strings.Contains(got, "Scope: intractive") {
		t.Errorf("Caveat() = %q, want the scope named even when no wording is registered", got)
	}
}

// Every scope a detector can declare has reader-facing wording; a missing one silently falls back
// to a constant written for a switch statement.
func TestEveryScopeHasReaderFacingWording(t *testing.T) {
	for _, s := range []string{Interactive, SubAgent, Programmatic, Unstated} {
		if scopeCovers[s] == "" {
			t.Errorf("scope %q has no wording, so its caveat would print the constant", s)
		}
	}
}
