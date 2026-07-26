package server

import (
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

// TestValidateRecordEnforcesRecordInvariants guards the two whole-record contracts no
// per-field bound can see. The purpose buckets split ToolCalls -- both parsers' fuzz tests
// assert they sum to it -- and the team dashboard reads that ratio as coverage, so a pushed
// split disagreeing with the count suppresses the partial-coverage caveat and presents an
// incomplete split as a complete one. Sidechain is a flag, not a count.
func TestValidateRecordEnforcesRecordInvariants(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*usage.Record)
		wantErr bool
	}{
		{"split sums to tool calls", func(r *usage.Record) {
			r.ToolCalls, r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther = 10, 4, 2, 2, 1, 1
		}, false},
		{"all-zero split with tool calls", func(r *usage.Record) { r.ToolCalls = 12 }, false},
		{"neither counted", func(r *usage.Record) { r.ToolCalls, r.ToolReads = 0, 0 }, false},
		{"split disagrees with tool calls", func(r *usage.Record) {
			r.ToolCalls, r.ToolReads = 10, 4
		}, true},
		{"split without tool calls", func(r *usage.Record) { r.ToolReads = 5000 }, true},
		{"sidechain flag set", func(r *usage.Record) { r.Sidechain = 1 }, false},
		{"sidechain is not a flag", func(r *usage.Record) { r.Sidechain = 2 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newValidRecord()
			tt.mutate(&r)
			if err := validateRecord(&r); (err != nil) != tt.wantErr {
				t.Fatalf("validateRecord = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
