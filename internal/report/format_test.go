package report

import "testing"

func TestFormatCommas(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{7, "7"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{12345678, "12,345,678"},
	}
	for _, tt := range tests {
		if got := formatCommas(tt.n); got != tt.want {
			t.Fatalf("formatCommas(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
