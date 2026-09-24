package docs

import "strings"

// SitemapFile is the crawl map search engines read. It is generated from the guide list, so a
// guide is listed the day it is published and never after it is gone.
const SitemapFile = "site/sitemap.xml"

// Sitemap lists every page the site serves: the overview, each guide and the reference. The
// /reference redirect is left out, since a crawler should index where it points. There is no
// lastmod: the committed file must equal what this prints, and a date would change on every run.
func Sitemap(guides []Guide) []byte {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, u := range sitemapURLs(guides) {
		b.WriteString("  <url><loc>" + Host + u + "</loc></url>\n")
	}
	b.WriteString("</urlset>\n")
	return []byte(b.String())
}

func sitemapURLs(guides []Guide) []string {
	urls := []string{"/"}
	seen := map[string]bool{"/": true}
	add := func(u string) {
		if !seen[u] && !strings.HasPrefix(u, "#") {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	for i := range guides {
		add(guides[i].URL)
	}
	add(ReferenceURL)
	return urls
}
