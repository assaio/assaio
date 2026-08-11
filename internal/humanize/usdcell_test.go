package humanize

import "testing"

func TestUSDCell(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want string
	}{
		{"zero is a real zero", 0, "0.00"},
		{"cents from a dollar up", 4.7230, "4.72"},
		{"a six-figure total groups", 107640.1234, "107,640.12"},
		{"exactly one dollar", 1, "1.00"},
		{"two significant digits below a dollar", 0.1783, "0.18"},
		{"a tenth of a cent keeps its digits", 0.0032, "0.0032"},
		{"rounding stays inside four decimals", 0.00012345, "0.0001"},
		{"below the floor states a bound, never zero", 0.00001, "<0.0001"},
		{"a negative delta keeps its sign", -12.5, "-12.50"},
		{"a small negative delta keeps its digits", -0.0032, "-0.0032"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := USDCell(tt.in); got != tt.want {
				t.Fatalf("USDCell(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestThousands(t *testing.T) {
	tests := []struct {
		in   float64
		prec int
		want string
	}{
		{12345, 0, "12,345"},
		{999, 0, "999"},
		{1000, 0, "1,000"},
		{107640.1, 2, "107,640.10"},
		{-12345.5, 1, "-12,345.5"},
		{0, 2, "0.00"},
	}
	for _, tt := range tests {
		if got := Thousands(tt.in, tt.prec); got != tt.want {
			t.Fatalf("Thousands(%v, %d) = %q, want %q", tt.in, tt.prec, got, tt.want)
		}
	}
}

func TestInt(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{7, "7"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{12345678, "12,345,678"},
		{-12345, "-12,345"},
		{9007199254740993, "9,007,199,254,740,993"},
	}
	for _, tt := range tests {
		if got := Int(tt.in); got != tt.want {
			t.Errorf("Int(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
