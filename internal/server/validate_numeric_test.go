package server

import (
	"reflect"
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

// TestValidateRecordBoundsEveryNumericField walks usage.Record by reflection so a numeric
// field added later cannot reach the shared store unbounded: an unchecked negative renders
// impossible percentages in the team report, and an unchecked overflow-magnitude value
// makes SUM() fail for every member at once. Migration 0002 added nine columns and the
// bounds array was not extended with them; this test is what makes that a build failure
// rather than a silent hole.
func TestValidateRecordBoundsEveryNumericField(t *testing.T) {
	typ := reflect.TypeOf(usage.Record{})
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Type.Kind() != reflect.Int64 {
			continue
		}
		for _, tc := range []struct {
			name  string
			value int64
		}{
			{"negative", -1},
			{"overflow magnitude", maxFieldValue + 1},
		} {
			t.Run(field.Name+"/"+tc.name, func(t *testing.T) {
				r := newValidRecord()
				reflect.ValueOf(&r).Elem().FieldByName(field.Name).SetInt(tc.value)
				if err := validateRecord(&r); err == nil {
					t.Fatalf("%s = %d passed validation; every numeric field must be bounded", field.Name, tc.value)
				}
			})
		}
	}
}
