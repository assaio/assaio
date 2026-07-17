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
var syncHTTPClient = &http.Client{Timeout: syncHTTPTimeout}

// syncPushRequest is POST /v1/usage's request body (see internal/server's usagePush).
type syncPushRequest struct {
	Member  string         `json:"member"`
	Records []usage.Record `json:"records"`
}

// syncPushResponse is POST /v1/usage's response body (see internal/server's
// usagePushResult).
type syncPushResponse struct {
	Inserted int `json:"inserted"`
	Received int `json:"received"`
}

// pushUsage POSTs recs to serverURL's /v1/usage endpoint as member, authenticating with
// token.
func pushUsage(ctx context.Context, serverURL, token, member string, recs []usage.Record) (syncPushResponse, error) {
	body, err := json.Marshal(syncPushRequest{Member: member, Records: recs})
	if err != nil {
		return syncPushResponse{}, err
	}

	url := strings.TrimSuffix(serverURL, "/") + "/v1/usage"
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
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, maxSyncErrorBodyBytes))
		return syncPushResponse{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(data))
	}

	var result syncPushResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return syncPushResponse{}, fmt.Errorf("decode server response: %w", err)
	}
	return result, nil
}
