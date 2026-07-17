package server

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// bearerPrefix precedes the token in a request's Authorization header.
const bearerPrefix = "Bearer "

// authorized reports whether r carries exactly the expected bearer token, compared in
// constant time so response timing can't leak how many prefix bytes matched. A
// misconfigured empty server token always denies -- it must never be treated as "no
// auth required".
func authorized(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, bearerPrefix) {
		return false
	}
	got := strings.TrimPrefix(h, bearerPrefix)
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}
