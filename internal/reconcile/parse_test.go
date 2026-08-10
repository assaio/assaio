package reconcile

import "testing"

func TestParseDay(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"plain date", "2026-08-09", "2026-08-09", true},
		{"rfc3339 normalizes to UTC", "2026-08-09T23:30:00-05:00", "2026-08-10", true},
		{"space separated", "2026-08-09 14:00:00", "2026-08-09", true},
		{"slashes", "2026/08/09", "2026-08-09", true},
		{"padded", "  2026-08-09  ", "2026-08-09", true},
		{"an unrecognized shape is skipped, never guessed", "09/08/2026", "", false},
		{"empty", "", "", false},
		{"not a date", "yesterday", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseDay(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseDay(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseMoney(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want float64
		ok   bool
	}{
		{"plain", "4.72", 4.72, true},
		{"dollar sign", "$4.72", 4.72, true},
		{"thousands", "$107,640.12", 107640.12, true},
		{"trailing currency", "4.72 USD", 4.72, true},
		{"parenthesised credit is negative", "(12.50)", -12.5, true},
		{"leading minus", "-12.50", -12.5, true},
		{"empty", "", 0, false},
		{"not a number", "n/a", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseMoney(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseMoney(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseCount(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int64
		ok   bool
	}{
		{"plain", "16400", 16400, true},
		{"thousands", "1,657,000", 1657000, true},
		{"whole float", "16400.0", 16400, true},
		{"a fractional count is not a count", "16400.5", 0, false},
		{"empty means the export stated none", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseCount(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseCount(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}
