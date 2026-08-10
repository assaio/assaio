package reconcile

import "testing"

func TestBindColumns(t *testing.T) {
	tests := []struct {
		name      string
		headers   []string
		overrides map[string]string
		wantDay   string
		wantCost  string
		wantModel string
		wantErr   bool
	}{
		{
			name:    "aliases bind without help",
			headers: []string{"date", "model", "amount"},
			wantDay: "date", wantCost: "amount", wantModel: "model",
		},
		{
			name:    "separators and case fold",
			headers: []string{"Usage Date", "Cost USD"},
			wantDay: "Usage Date", wantCost: "Cost USD",
		},
		{
			name:      "an override beats an alias",
			headers:   []string{"date", "amount", "charge"},
			overrides: map[string]string{FieldCost: "charge"},
			wantDay:   "date", wantCost: "charge",
		},
		{
			name:    "no date column is an error, not a guess",
			headers: []string{"model", "amount"},
			wantErr: true,
		},
		{
			name:    "no money column is an error",
			headers: []string{"date", "model"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, cols, err := bindColumns(tt.headers, tt.overrides)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want an error naming the columns it saw")
				}
				return
			}
			if err != nil {
				t.Fatalf("bindColumns: %v", err)
			}
			if b.Day != tt.wantDay || b.Cost != tt.wantCost || b.Model != tt.wantModel {
				t.Fatalf("bound day=%q cost=%q model=%q; want %q/%q/%q", b.Day, b.Cost, b.Model, tt.wantDay, tt.wantCost, tt.wantModel)
			}
			if _, ok := cols[FieldDay]; !ok {
				t.Fatal("day column index missing")
			}
		})
	}
}

func TestParseMap(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    map[string]string
		wantErr bool
	}{
		{"empty is no overrides", "", map[string]string{}, false},
		{"one pair", "cost=amount", map[string]string{FieldCost: "amount"}, false},
		{"several, spaced", " cost=amount , day=usage_date ", map[string]string{FieldCost: "amount", FieldDay: "usage_date"}, false},
		{"an unknown field is rejected, never ignored", "spend=amount", nil, true},
		{"a malformed pair is rejected", "cost", nil, true},
		{"an empty column is rejected", "cost=", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMap(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMap: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseMap(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Fatalf("ParseMap(%q)[%s] = %q, want %q", tt.in, k, got[k], v)
				}
			}
		})
	}
}
