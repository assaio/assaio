package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FetchLimits bound one read of a metrics endpoint. They are the whole security posture of
// this feature: it points a client at a URL a user supplies, so time, size and redirects are
// budgets rather than defaults somebody else chose.
type FetchLimits struct {
	// Timeout bounds the whole request, connection through last byte.
	Timeout time.Duration
	// MaxBytes bounds what is read from the response, whatever it claims to be.
	MaxBytes int64
	// MaxRedirects bounds how far a redirect chain may lead. Zero forbids redirects entirely.
	MaxRedirects int
}

// DefaultFetchLimits are sized for an exporter on a local network.
func DefaultFetchLimits() FetchLimits {
	return FetchLimits{Timeout: 5 * time.Second, MaxBytes: 8 << 20, MaxRedirects: 2}
}

// ErrTooLarge is returned when an endpoint's body exceeds the budget. It is an error rather
// than a truncated read: a partially read exposition would make every metric it did not reach
// look absent, which is the one mistake this feature cannot afford.
var ErrTooLarge = errors.New("response exceeds the size limit")

// Fetch reads a metrics endpoint under lim. It is a GET and nothing else -- no header a caller
// could inject, no body, no credential -- because an inspection that could carry a secret would
// need a threat model this experiment does not have.
func Fetch(ctx context.Context, url string, lim FetchLimits) ([]byte, error) {
	if lim.Timeout <= 0 {
		lim = DefaultFetchLimits()
	}
	ctx, cancel := context.WithTimeout(ctx, lim.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building the request: %w", err)
	}
	req.Header.Set("Accept", "text/plain")

	client := &http.Client{
		Timeout: lim.Timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) > lim.MaxRedirects {
				return fmt.Errorf("more than %d redirect(s); a metrics endpoint that redirects this far is not the one you meant", lim.MaxRedirects)
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s from %s", resp.Status, url)
	}
	// One byte past the budget is read on purpose: it is what tells a body that is exactly at
	// the limit from one that is over it.
	body, err := io.ReadAll(io.LimitReader(resp.Body, lim.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > lim.MaxBytes {
		return nil, fmt.Errorf("%w (%d bytes)", ErrTooLarge, lim.MaxBytes)
	}
	return body, nil
}
