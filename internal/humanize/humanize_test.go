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
		{999_999, "1000.0K"},
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
