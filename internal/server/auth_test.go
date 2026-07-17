package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorized(t *testing.T) {
	tests := []struct {
		name   string
		header string
		token  string
		want   bool
	}{
		{"valid bearer", "Bearer secret", "secret", true},
		{"wrong token", "Bearer wrong", "secret", false},
		{"missing header", "", "secret", false},
		{"missing bearer prefix", "secret", "secret", false},
		{"empty server token always denies", "Bearer ", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			if got := authorized(req, tt.token); got != tt.want {
				t.Fatalf("authorized() = %v, want %v", got, tt.want)
			}
		})
	}
}
