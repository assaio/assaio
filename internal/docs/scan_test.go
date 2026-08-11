package docs

import (
	"strings"
	"testing"
)

// The promise a covered set makes is "this list enumerates them". A claim that sits somewhere
// else on the page, or inside a comment, does not keep it -- and the first version of this
// checker accepted both, which would have let the `digest` regression recur with a green build.
func TestCoverageIsScopedToTheElementThatPromisesIt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "a claim inside the covering element satisfies it",
			content: `<dl data-covers="commands">
				<dt data-claim="command.digest">digest</dt>
				<dt data-claim="command.signals">signals</dt></dl>`,
		},
		{
			name: "a claim elsewhere on the page does not",
			content: `<dl data-covers="commands"><dt data-claim="command.digest">digest</dt></dl>
				<p><code data-claim="command.signals">signals</code> is documented over here</p>`,
			want: `never claims "command.signals" inside that element`,
		},
		{
			name: "a commented-out claim does not, even inside the element",
			content: `<dl data-covers="commands">
				<dt data-claim="command.digest">digest</dt>
				<!-- <dt data-claim="command.signals">signals</dt> --></dl>`,
			want: `never claims "command.signals" inside that element`,
		},
		{
			name: "a nested element of the same name does not end the enclosure early",
			content: `<div data-covers="commands">
				<div><span data-claim="command.digest">digest</span></div>
				<span data-claim="command.signals">signals</span></div>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := CheckClaims(fixture(), "page.html", tt.content)
			if tt.want == "" {
				if len(problems) > 0 {
					t.Fatalf("expected no problem, got %v", problems)
				}
				return
			}
			for _, p := range problems {
				if strings.Contains(p.Text, tt.want) {
					return
				}
			}
			t.Fatalf("expected a problem containing %q, got %v", tt.want, problems)
		})
	}
}

// A count nobody can read is not a count that was checked.
func TestACountClaimOnUnreadableMarkupFails(t *testing.T) {
	problems := CheckClaims(fixture(), "page.html", `<span data-claim="signals.count"`)
	if len(problems) == 0 {
		t.Fatal("expected a problem: no element states the number")
	}
	if !strings.Contains(problems[0].Text, "no element carrying it states a number") {
		t.Errorf("problem = %q", problems[0].Text)
	}
}
