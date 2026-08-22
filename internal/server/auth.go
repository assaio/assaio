package server

import (
	"net/http"
	"strings"
)

// bearerPrefix precedes the token in a request's Authorization header.
const bearerPrefix = "Bearer "

// bearer extracts the presented token, reporting whether the header carried one at all. The
// two answers are kept apart from whether it is *correct*: a missing header and a wrong secret
// are the same response to a client and different lines in a log.
func bearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, bearerPrefix) {
		return "", false
	}
	return strings.TrimPrefix(h, bearerPrefix), true
}

// authorizedReader reports whether r may read team-wide data. Any configured secret grants a
// read: this server has no roles yet, and inventing one here would be a contract the rest of
// the product does not carry. What it does not do any more is grant a read to nobody at all.
func (s *Server) authorizedReader(r *http.Request) bool {
	presented, present := bearer(r)
	return present && s.authenticated(presented)
}
