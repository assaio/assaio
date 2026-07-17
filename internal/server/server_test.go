package server

import (
	"net/http"
	"testing"
)

// TestNewHTTPServerSetsAllTimeouts proves every socket timeout is wired -- before this
// fix, only ReadHeaderTimeout was set, leaving a stalled client (or one that trickles a
// request/response) able to hold a connection open indefinitely.
func TestNewHTTPServerSetsAllTimeouts(t *testing.T) {
	srv := newHTTPServer(":0", http.NewServeMux())
	if srv.ReadHeaderTimeout <= 0 {
		t.Error("ReadHeaderTimeout must be set")
	}
	if srv.ReadTimeout <= 0 {
		t.Error("ReadTimeout must be set")
	}
	if srv.WriteTimeout <= 0 {
		t.Error("WriteTimeout must be set")
	}
	if srv.IdleTimeout <= 0 {
		t.Error("IdleTimeout must be set")
	}
	if srv.ReadHeaderTimeout > srv.ReadTimeout {
		t.Error("ReadHeaderTimeout must not exceed ReadTimeout")
	}
}
