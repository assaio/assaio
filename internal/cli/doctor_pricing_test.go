package cli

import (
	"bytes"
	"errors"
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

func TestDoctorPricingLine(t *testing.T) {
	tests := []struct {
		name     string
		retained int
		err      error
		want     string
	}{
		{
			"none retained", 0, nil,
			"pricing:      4000 models, snapshot 2026-09-23 (refresh ships with releases)",
		},
		{
			"some retained", 73, nil,
			"pricing:      4000 models, snapshot 2026-09-23, 73 models dropped by LiteLLM priced at last listed prices (refresh ships with releases)",
		},
		{
			"table failed to load", 0, errors.New("unexpected EOF"),
			"pricing:      Price table parse failed; all costs are unpriced: unexpected EOF",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doctorPricingLine(4000, tt.retained, "2026-09-23", tt.err); got != tt.want {
				t.Errorf("line = %q\nwant   %q", got, tt.want)
			}
		})
	}
}
