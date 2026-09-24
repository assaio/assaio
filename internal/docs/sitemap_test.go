package docs

import (
	"slices"
	"testing"
)

func TestSitemapURLs(t *testing.T) {
	guides := []Guide{{URL: "/docs"}, {URL: "/docs/evidence"}, {URL: "#signals"}, {URL: "/docs/evidence"}}
	want := []string{"/", "/docs", "/docs/evidence", ReferenceURL}
	if got := sitemapURLs(guides); !slices.Equal(got, want) {
		t.Fatalf("sitemapURLs = %v, want %v", got, want)
	}
}
