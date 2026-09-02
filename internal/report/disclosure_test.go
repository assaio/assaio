package report

import (
	"strings"
	"testing"
)

// A window can be short for two reasons at once, and only one of them is a release away. The
// switch answers Missing() first, so a store holding an unlisted model *and* rows from a source
// that publishes no token counter used to print the refresh promise alone -- the untokened
// explanation was reachable only where nothing else was unpriced at all.
func TestUnpricedDisclosureNamesBothReasonsWhenBothHold(t *testing.T) {
	const refresh = "a refreshed table ships with each release"
	const noCounter = "no price-table refresh changes it"
	for _, tc := range []struct {
		name          string
		u             Unpriced
		wants, absent []string
	}{
		{
			name:  "a model the table has yet to carry",
			u:     Unpriced{Tokens: 455, Total: 1000, Rows: 2},
			wants: []string{refresh}, absent: []string{noCounter},
		},
		{
			name:  "only rows with nothing to price",
			u:     Unpriced{Total: 1000, Rows: 3, Untokened: 3},
			wants: []string{noCounter}, absent: []string{refresh},
		},
		{
			name: "both, which is the case the switch used to hide",
			u:    Unpriced{Tokens: 455, Total: 1000, Rows: 5, Untokened: 3},
			// The count is the untokened rows, never the share: they carry no tokens, so they are
			// absent from the percentage the first clause states rather than part of it.
			wants: []string{refresh, noCounter, "3 of the unpriced rows"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := UnpricedDisclosure(&tc.u, "the tokens in view")
			for _, want := range tc.wants {
				if !strings.Contains(got, want) {
					t.Errorf("disclosure = %q, want it to say %q", got, want)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(got, absent) {
					t.Errorf("disclosure = %q, want it not to say %q", got, absent)
				}
			}
		})
	}
}
