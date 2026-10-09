package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// syncHTTPTimeout bounds one push request end-to-end (connect, send, and receive), so a
// hung or unreachable server can't leave `sync` blocked forever. Generous relative to
// the team server's own write timeout (internal/server's writeTimeout) so a
// legitimately large first-time sync isn't cut off before the server would give up on
// it; Ctrl-C (see runSync's signal.NotifyContext) aborts sooner if the caller doesn't
// want to wait this long.
const syncHTTPTimeout = 120 * time.Second

// maxSyncErrorBodyBytes caps how much of a non-200 response body pushUsage reads back
// into an error message, so a misbehaving or malicious server can't make `sync` buffer
// an unbounded response.
const maxSyncErrorBodyBytes = 64 << 10

// syncHTTPClient is the client pushUsage uses -- never http.DefaultClient, which has no
// timeout and would let an unresponsive server hang `sync` indefinitely.
var syncHTTPClient = &http.Client{
	Timeout:       syncHTTPTimeout,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
}

// syncPushResponse is POST /v2/usage's response body.
type syncPushResponse struct {
	Protocol int `json:"protocol"`
	Inserted int `json:"inserted"`
	Received int `json:"received"`
}

// pushUsage never falls back to v1: the old server would key the same rows under a
// different member prefix, duplicating a team's usage.
func pushUsage(ctx context.Context, serverURL, token, member string, recs []usage.Record) (syncPushResponse, error) {
	body, err := json.Marshal(newSyncPushRequestV2(member, recs))
	if err != nil {
		return syncPushResponse{}, err
	}

	url := strings.TrimSuffix(serverURL, "/") + "/v2/usage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return syncPushResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := syncHTTPClient.Do(req)
	if err != nil {
		return syncPushResponse{}, fmt.Errorf("reach server %s: %w", serverURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return syncPushResponse{}, fmt.Errorf("server rejected the token (401) at %s", serverURL)
	}
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusUpgradeRequired {
		return syncPushResponse{}, fmt.Errorf("server at %s does not accept sync protocol v2; upgrade the server and migrate its identity before retrying", serverURL)
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return syncPushResponse{}, fmt.Errorf("server redirected sync protocol v2 (%d); refusing to send records to another endpoint", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, maxSyncErrorBodyBytes))
		return syncPushResponse{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(data))
	}

	var result syncPushResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxSyncErrorBodyBytes)).Decode(&result); err != nil {
		return syncPushResponse{}, fmt.Errorf("decode server response: %w", err)
	}
	if result.Protocol != 2 || result.Received != len(recs) || result.Inserted < 0 || result.Inserted > result.Received {
		return syncPushResponse{}, fmt.Errorf("server response does not confirm sync protocol v2 and %d received records", len(recs))
	}
	return result, nil
}
