package humanize

import "testing"

func TestPercentAt(t *testing.T) {
	tests := []struct {
		name     string
		share    float64
		decimals int
		want     string
	}{
		{"exact zero stays zero", 0, 0, "0%"},
		{"a real but tiny share never reads absent", 0.00157, 0, "<1%"},
		{"the 102-refusal case", 102.0 / 65098.0, 0, "<1%"},
		{"same case at one decimal", 102.0 / 65098.0, 1, "0.2%"},
		{"tiny at one decimal", 0.0002, 1, "<0.1%"},
		{"a real remainder is never rounded away", 0.996, 0, ">99%"},
		{"whole is whole", 1, 0, "100%"},
		{"whole at one decimal", 1, 1, "100.0%"},
		{"just under whole at one decimal", 0.99995, 1, ">99.9%"},
		{"ordinary share", 0.43, 0, "43%"},
		{"ordinary share at one decimal", 0.4321, 1, "43.2%"},
	}
	for _, tt := range tests {
		if got := PercentAt(tt.share, tt.decimals); got != tt.want {
			t.Errorf("%s: PercentAt(%v, %d) = %q, want %q", tt.name, tt.share, tt.decimals, got, tt.want)
		}
	}
}

func TestPercentOrDash(t *testing.T) {
	tests := []struct {
		num, den int64
		decimals int
		want     string
	}{
		{0, 0, 0, "—"},
		{5, 0, 1, "—"},
		{102, 65098, 0, "<1%"},
		{102, 65098, 1, "0.2%"},
		{1, 4, 0, "25%"},
	}
	for _, tt := range tests {
		if got := PercentOrDash(tt.num, tt.den, tt.decimals); got != tt.want {
			t.Errorf("PercentOrDash(%d, %d, %d) = %q, want %q", tt.num, tt.den, tt.decimals, got, tt.want)
		}
	}
}

// TestPercentEdgesAreSymmetric is B127's sibling: an exact 0.5% fell through to the
// default and printed "0%" -- the precise rounding this function exists to refuse -- while
// an exact 99.5% was already caught by the upper guard's >=.
func TestPercentEdgesAreSymmetric(t *testing.T) {
	if got := Percent(0.005); got != "<1%" {
		t.Errorf("Percent(0.005) = %q, want %q", got, "<1%")
	}
	if got := Percent(0.995); got != ">99%" {
		t.Errorf("Percent(0.995) = %q, want %q", got, ">99%")
	}
}
