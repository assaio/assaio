package humanize

import "testing"

func TestCount(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1_000, "1.0K"},
		{2_300, "2.3K"},
		{999_999, "1.0M"},
		{1_500_000, "1.5M"},
		{33_400_000_000, "33.4B"},
	}
	for _, tt := range tests {
		if got := Count(tt.in); got != tt.want {
			t.Errorf("Count(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestUSD(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0.00"},
		{1.5, "1.50"},
		{99.994, "99.99"},
		{195, "195"},
		{9_999, "9999"},
		{26_004, "26.0K"},
	}
	for _, tt := range tests {
		if got := USD(tt.in); got != tt.want {
			t.Errorf("USD(%g) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{35_131_392, "33.5 MB"},
		{4_831_838_208, "4.5 GB"},
	}
	for _, tt := range tests {
		if got := Bytes(tt.in); got != tt.want {
			t.Errorf("Bytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestCountPicksItsUnitAfterRounding is B127: the unit was chosen from the raw value, so a
// count that rounds up into the next unit printed "1000.0M" instead of "1.0B". Real
// cache-read totals sit exactly in that band.
func TestCountPicksItsUnitAfterRounding(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{999, "999"},
		{999_950, "1.0M"},
		{999_999_999, "1.0B"},
		{1_500, "1.5K"},
		{33_400_000_000, "33.4B"},
	}
	for _, tt := range tests {
		if got := Count(tt.n); got != tt.want {
			t.Errorf("Count(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestBytesPicksItsUnitAfterRounding(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{1023, "1023 B"},
		{1_048_575, "1.0 MB"},
		{35_131_392, "33.5 MB"},
	}
	for _, tt := range tests {
		if got := Bytes(tt.n); got != tt.want {
			t.Errorf("Bytes(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// TestUSDCompactNeverPrintsAFabricatedZero is B131: $12 across 30 active days is $0.40 and
// printed "$0", which reads as no cost at all.
func TestUSDCompactNeverPrintsAFabricatedZero(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{0, "$0"},
		{0.0004, "<$0.01"},
		{0.4, "$0.40"},
		{17, "$17"},
		{999, "$999"},
		{1000, "$1.0K"},
		{31500, "$31.5K"},
		{999_999.9, "$1.0M"},
		{1_500_000, "$1.5M"},
	}
	for _, tt := range tests {
		if got := USDCompact(tt.v); got != tt.want {
			t.Errorf("USDCompact(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}
