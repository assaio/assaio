package plugin

import (
	"encoding/json"
	"fmt"
)

// protocolVersion is the only handshake version assaio understands.
const protocolVersion = 1

// handshake is line 1 of a plugin's stdout, identifying the protocol version and
// confirming the plugin's own name.
type handshake struct {
	Protocol int    `json:"assaio_plugin"`
	Tool     string `json:"tool"`
}

// parseHandshake validates line against the expected plugin name and protocol version.
func parseHandshake(line []byte, wantName string) error {
	var h handshake
	if err := json.Unmarshal(line, &h); err != nil {
		return fmt.Errorf("invalid handshake JSON: %w", err)
	}
	if h.Protocol != protocolVersion {
		return fmt.Errorf("unsupported protocol version %d (want %d)", h.Protocol, protocolVersion)
	}
	if h.Tool != wantName {
		return fmt.Errorf("handshake tool %q does not match configured name %q", h.Tool, wantName)
	}
	return nil
}
