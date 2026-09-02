package usage

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCheckTimestamp(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		ts      time.Time
		wantErr bool
	}{
		{"now", now, false},
		{"the floor itself", TimestampFloor, false},
		{"a day before now", now.AddDate(0, 0, -1), false},
		{"within the skew", now.Add(FutureSkew - time.Hour), false},
		{"the zero time", time.Time{}, true},
		{"before the floor", TimestampFloor.Add(-time.Second), true},
		{"past the skew", now.Add(FutureSkew + time.Hour), true},
		{"year 9999", time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTimestamp(tt.ts, now)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTimestamp(%s) err = %v, wantErr %v", tt.ts, err, tt.wantErr)
			}
		})
	}
}

// The reason is asserted rather than the rejection alone: this gate stands at the entrance to the
// store and to the team server, and one that rejects for the wrong reason is one that will reject
// the wrong record later. The rows with two violations at once pin which reason wins.
func TestCheckCounts(t *testing.T) {
	tests := []struct {
		name string
		rec  Record
		// want is a substring of the rejection reason; "" means the record must be accepted.
		want string
	}{
		{"empty record", Record{}, ""},
		{"ordinary turn", Record{InputTokens: 100, OutputTokens: 50, ReasoningTokens: 20}, ""},
		{"reasoning equal to output", Record{OutputTokens: 50, ReasoningTokens: 50}, ""},
		{"cache tier equal to its whole", Record{CacheWriteTokens: 10, CacheWrite1hTokens: 10}, ""},
		{"the cap itself", Record{InputTokens: MaxCount}, ""},

		{"negative token", Record{InputTokens: -1}, "negative count field"},
		{"the most negative int64", Record{OutputTokens: math.MinInt64}, "negative count field"},
		{"overflow magnitude", Record{InputTokens: MaxCount + 1}, "count field exceeds 1000000000"},
		{"the largest int64", Record{CacheReadTokens: math.MaxInt64}, "count field exceeds 1000000000"},

		{"cache tier above its whole", Record{CacheWriteTokens: 10, CacheWrite1hTokens: 11}, "cache_write_1h 11 exceeds cache_write 10"},
		{"a cache tier with no write behind it", Record{CacheWrite1hTokens: 1}, "cache_write_1h 1 exceeds cache_write 0"},
		{"reasoning above output", Record{OutputTokens: 50, ReasoningTokens: 51}, "reasoning_tokens 51 exceeds output_tokens 50"},
		{"reasoning with no output behind it", Record{ReasoningTokens: 1}, "reasoning_tokens 1 exceeds output_tokens 0"},

		// Negative values also satisfy "subset above its whole"; the magnitude loop runs first, so
		// the record is rejected for the impossible number rather than for the relation it broke.
		{"a negative subset under a negative whole", Record{CacheWriteTokens: -2, CacheWrite1hTokens: -1}, "negative count field"},
		{"an over-cap subset under a zero whole", Record{CacheWrite1hTokens: MaxCount + 1}, "count field exceeds 1000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckCounts(&tt.rec)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("CheckCounts rejected a valid record: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("CheckCounts accepted a record it must reject, want %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("CheckCounts rejected with %q, want the reason to carry %q", err, tt.want)
			}
		})
	}
}

// Every count on the record is inside the gate. bounds.go lists the fields by hand, so a field
// added to Record and forgotten there would cross both boundaries unbounded and land in a SUM().
// Read off the struct rather than listed again here, which would only move the same omission.
func TestEveryCountFieldOnTheRecordIsBounded(t *testing.T) {
	fields := countFields()
	if len(fields) == 0 {
		t.Fatal("found no int64 fields on Record, so nothing below was asserted")
	}
	for _, name := range fields {
		t.Run(name, func(t *testing.T) {
			negative := withCount(name, -1)
			if err := CheckCounts(&negative); err == nil {
				t.Errorf("%s = -1 was accepted", name)
			} else if !strings.Contains(err.Error(), "negative count field") {
				t.Errorf("%s = -1 was rejected as %q, want the negative-count reason", name, err)
			}
			over := withCount(name, MaxCount+1)
			if err := CheckCounts(&over); err == nil {
				t.Errorf("%s = %d was accepted", name, int64(MaxCount+1))
			} else if !strings.Contains(err.Error(), "count field exceeds") {
				t.Errorf("%s = %d was rejected as %q, want the magnitude reason", name, int64(MaxCount+1), err)
			}
		})
	}
}

// What MaxCount is for: a record the gate accepted cannot, on its own, put a stored total near an
// int64 overflow. Maxed out in every count it still leaves room for hundreds of millions of them
// in one sum, which is the headroom a SUM() over a real store needs.
func TestAnAcceptedRecordCannotOverflowAnInt64Sum(t *testing.T) {
	fields := countFields()
	maxed := Record{}
	rv := reflect.ValueOf(&maxed).Elem()
	for _, name := range fields {
		rv.FieldByName(name).SetInt(MaxCount)
	}
	if err := CheckCounts(&maxed); err != nil {
		t.Fatalf("a record at the cap in every field was rejected: %v", err)
	}

	sum := int64(len(fields)) * MaxCount
	if sum <= 0 {
		t.Fatalf("the caps sum to %d, which has already wrapped", sum)
	}
	if headroom := math.MaxInt64 / sum; headroom < 1_000_000 {
		t.Errorf("one accepted record is within %dx of overflowing an int64 sum", headroom)
	}
}

// A cache read is billed alongside input_tokens rather than out of it (internal/pricing prices the
// two additively), so 472,924 cache-read tokens against 2 fresh input ones is the ordinary shape
// of a long session, not a contradiction. Recorded because it reads like one: a subset check
// between the two would reject most of a real store.
func TestCheckCountsAcceptsALargeCacheReadAgainstASmallInput(t *testing.T) {
	r := Record{InputTokens: 2, OutputTokens: 63, CacheReadTokens: 472_924, CacheWriteTokens: 740}
	if err := CheckCounts(&r); err != nil {
		t.Fatalf("a cache-hit turn was rejected: %v", err)
	}
}

// countFields is every int64 field on Record: the numbers that reach a stored SUM(), and therefore
// exactly the set CheckCounts must bound.
func countFields() []string {
	var names []string
	rt := reflect.TypeOf(Record{})
	for i := range rt.NumField() {
		if f := rt.Field(i); f.Type.Kind() == reflect.Int64 {
			names = append(names, f.Name)
		}
	}
	return names
}

func withCount(name string, v int64) Record {
	var r Record
	reflect.ValueOf(&r).Elem().FieldByName(name).SetInt(v)
	return r
}
