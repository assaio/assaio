package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/report"
)

// A store can be short on price for two reasons at once, and the switch answers Missing() first.
// Where both hold, the reader used to be told only that a refreshed table ships with each
// release -- a fix that closes part of the gap, presented as the fix for all of it.
func TestUnpricedSectionNamesRowsNoRefreshCanPrice(t *testing.T) {
	for _, tc := range []struct {
		name          string
		unpriced      report.Unpriced
		wants, absent []string
	}{
		{
			name:     "a model the table has yet to carry",
			unpriced: report.Unpriced{Tokens: 455, Total: 1000, Rows: 2},
			wants:    []string{"Upgrade assaio for a refreshed price table"},
			absent:   []string{"publish no token counter"},
		},
		{
			name:     "both reasons at once",
			unpriced: report.Unpriced{Tokens: 455, Total: 1000, Rows: 5, Untokened: 3},
			wants: []string{
				"Upgrade assaio for a refreshed price table",
				"3 of the unpriced row(s) publish no token counter at all, which no refresh changes",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			var out bytes.Buffer
			cmd.SetOut(&out)
			c := storeContents{Unpriced: tc.unpriced, Models: []string{"unknown-model"}, Window: "30d"}

			// maxShare 0 disables the --strict gate: this test is about what the reader is told,
			// not about what exits non-zero.
			if fail := unpricedSection(cmd, &c, 0); fail != "" {
				t.Fatalf("strict failure = %q, want none with the gate disabled", fail)
			}
			got := out.String()
			for _, want := range tc.wants {
				if !strings.Contains(got, want) {
					t.Errorf("unpriced section = %q, want it to say %q", got, want)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(got, absent) {
					t.Errorf("unpriced section = %q, want it not to say %q", got, absent)
				}
			}
		})
	}
}
