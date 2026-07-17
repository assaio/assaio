package report

import "testing"

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		sorted []float64
		p      float64
		want   float64
	}{
		{"empty", nil, 50, 0},
		{"single-p50", []float64{7}, 50, 7},
		{"single-p90", []float64{7}, 90, 7},
		{"odd-median", []float64{1, 2, 3, 4, 5}, 50, 3},
		{"odd-p90-interpolates", []float64{10, 20, 30, 40, 50}, 90, 46},
		{"p0-is-min", []float64{10, 20, 30}, 0, 10},
		{"p100-is-max", []float64{10, 20, 30}, 100, 30},
		{"even-median-interpolates", []float64{1, 2, 3, 4}, 50, 2.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.sorted, tt.p)
			if diff := got - tt.want; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("percentile(%v, %v) = %v, want %v", tt.sorted, tt.p, got, tt.want)
			}
		})
	}
}

func TestPercentileInt64(t *testing.T) {
	tests := []struct {
		name string
		vals []int64
		p    float64
		want int64
	}{
		{"empty", nil, 50, 0},
		{"single", []int64{5}, 50, 5},
		{"odd-median-unsorted", []int64{1, 3, 2}, 50, 2},
		{"even-median-rounds", []int64{1, 2, 3, 4}, 50, 3},
		{"p90-rounds", []int64{1, 2, 3, 4, 5}, 90, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percentileInt64(tt.vals, tt.p); got != tt.want {
				t.Fatalf("percentileInt64(%v, %v) = %d, want %d", tt.vals, tt.p, got, tt.want)
			}
		})
	}
}
