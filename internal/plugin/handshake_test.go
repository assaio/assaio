package plugin

import "testing"

func TestParseHandshake(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantName string
		wantErr  bool
	}{
		{"valid", `{"assaio_plugin":1,"tool":"demo"}`, "demo", false},
		{"wrong tool", `{"assaio_plugin":1,"tool":"other"}`, "demo", true},
		{"wrong version", `{"assaio_plugin":2,"tool":"demo"}`, "demo", true},
		{"invalid json", `not json`, "demo", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseHandshake([]byte(tt.line), tt.wantName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseHandshake() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
