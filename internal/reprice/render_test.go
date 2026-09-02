package reprice

import (
	"bytes"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/store"
)

// TestRenderProjectsEveryRefusal: the refusals are the reason this arithmetic is publishable
// at all, so they are rendered rather than documented. A renderer that dropped one would ship
// the figures without the conditions that make them true.
func TestRenderProjectsEveryRefusal(t *testing.T) {
	in := input([]store.UsageRow{
		row("big", 1_000_000, 100_000, 10_000_000, 1_000_000),
		row("small", 100_000, 10_000, 0, 0),
	}, 200)
	w := Compute(&in, Options{})

	var buf bytes.Buffer
	if err := RenderText(&buf, &w); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, line := range append(append([]string{}, w.Assumptions...), w.Refusals...) {
		if !strings.Contains(got, line) {
			t.Errorf("rendered block never states %q", line)
		}
	}
	for _, want := range []string{
		"(activity)", // the layer, stated the way every other surface states it
		"never quality for the same work",
		"No counterfactual",
		"rate limits and quotas are invisible",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered block is missing %q:\n%s", want, got)
		}
	}
}

// TestRenderStatesTheDirectionOfAMove: the columns report the change in what the window cost,
// so a route that saves money has to read as a smaller window rather than a positive delta.
func TestRenderStatesTheDirectionOfAMove(t *testing.T) {
	in := input([]store.UsageRow{
		row("big", 1_000_000, 100_000, 10_000_000, 1_000_000),
		row("small", 100_000, 10_000, 0, 0),
	}, 0)
	w := Compute(&in, Options{})

	var buf bytes.Buffer
	if err := RenderText(&buf, &w); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "-$15 (-79.4%)") {
		t.Errorf("route row does not state the move as a smaller window:\n%s", buf.String())
	}
}

// TestRenderNamesATargetItCouldNotPrice: a skipped target is invisible in a table that still
// has rows in it, and the rows that remain then read as the ranking the caller asked for.
func TestRenderNamesATargetItCouldNotPrice(t *testing.T) {
	in := input([]store.UsageRow{row("big", 1_000_000, 100_000, 10_000_000, 1_000_000)}, 0)
	w := Compute(&in, Options{Against: []string{"no-such-model", "small"}})

	var buf bytes.Buffer
	if err := RenderText(&buf, &w); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "small") {
		t.Fatalf("the priced target never reached the table:\n%s", got)
	}
	for _, want := range []string{"not priced", "no-such-model", "carries no rate for it"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered block is missing %q:\n%s", want, got)
		}
	}
}

func TestNames(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []string
		want string
	}{
		{"all of them", []string{"a", "b"}, "a, b"},
		{"one over the limit is cheaper to name", []string{"a", "b", "c", "d", "e"}, "a, b, c, d, e"},
		{"two over is worth a phrase", []string{"a", "b", "c", "d", "e", "f"}, "a, b, c, d (and 2 more)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := names(tc.in); got != tc.want {
				t.Errorf("names(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSignedPercentNeverReadsAsNoChange(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{-0.0001, "-<0.1%"},
		{0.0001, "+<0.1%"},
		{-0.7936, "-79.4%"},
		{0, "+0.0%"},
	} {
		if got := signedPercent(tc.in); got != tc.want {
			t.Errorf("signedPercent(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
