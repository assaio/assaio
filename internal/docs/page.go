package docs

import (
	"encoding/json"
	"strings"
)

// Host prefixes every link a generated page emits. Absolute rather than host-relative so the
// same file reads correctly opened straight from disk -- the posture the offline dashboard has,
// applied to the pages that document it.
const Host = "https://assaio.dev"

// pageSpec is one published page. Description feeds the meta tags a shared link renders from;
// Current marks which sidebar entry the reader is on. Outline feeds "On this page", Sequence the
// previous and next links; either left empty renders nothing.
type pageSpec struct {
	Title       string
	Description string
	URL         string
	Current     string
	Body        string
	Guides      []Guide
	Outline     []Heading
	Sequence    []Guide
	Landing     bool
}

// renderPage wraps a body in the shell every served page shares. One shell, so the reference
// and the guides cannot drift into looking like two different sites -- and so the reference is
// reached from inside the documentation rather than standing beside it.
func renderPage(p *pageSpec) []byte {
	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light dark">
<title>` + esc(p.Title) + ` — assaio</title>
<meta name="description" content="` + esc(p.Description) + `">
<link rel="canonical" href="https://assaio.dev` + p.URL + `">
<meta property="og:type" content="article">
<meta property="og:url" content="https://assaio.dev` + p.URL + `">
<meta property="og:title" content="` + esc(p.Title) + ` — assaio">
<meta property="og:description" content="` + esc(p.Description) + `">
<meta property="og:image" content="https://assaio.dev/og.png">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:image:alt" content="assaio logo, cost and output headline, subtitle, install commands and tool names">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:image" content="https://assaio.dev/og.png">
<link rel="icon" href="` + favicon + `">
<script type="application/ld+json">` + breadcrumbs(p) + `</script>
<style>` + pageStyle + `</style>
</head>
<body><div class="wrap">
<header class="top">
  <a class="brand" href="` + Host + `/"><svg viewBox="0 0 32 32" aria-hidden="true"><rect width="32" height="32" rx="8" fill="var(--signal)"/><path d="M9 22 16 10l7 12" stroke="var(--bg)" stroke-width="2.6" fill="none" stroke-linecap="round" stroke-linejoin="round"/></svg>assaio</a>
  <nav class="topnav">
    <a class="opt" href="` + Host + `/">Overview</a>
    <a href="` + Host + `/docs">Docs</a>
    <a href="` + Host + ReferenceURL + `">Reference</a>
    <a href="https://github.com/assaio/assaio">GitHub</a>
    <a class="opt" href="https://github.com/assaio/assaio/releases">Releases</a>
    <a class="menu" href="#docs-nav">Menu</a>
  </nav>
</header>
`)
	toc := tocHTML(p.Outline)
	layout, main := "layout", "doc"
	if toc != "" {
		layout += " with-toc"
	}
	if p.Landing {
		main += " landing"
	}
	b.WriteString(`<div class="` + layout + `">`)
	writeSidebar(&b, p)
	b.WriteString(toc + `<main class="` + main + `">` + p.Body + pagerHTML(p.Sequence, p.URL) + `</main>
</div>
<footer>
<p>Generated from the repository: every page here is rendered from the Markdown it is written in,
or from the binary's own registries, and a check fails the build when the two disagree. This page
names no release — which version is current is stated once, on the
<a href="https://github.com/assaio/assaio/releases">releases page</a>, where it cannot fall behind.
It loads nothing: no fonts, no analytics, no third-party requests.</p>
</footer>
</div></body></html>
`)
	return []byte(b.String())
}

func writeSidebar(b *strings.Builder, p *pageSpec) {
	b.WriteString(`<aside class="side" id="docs-nav"><nav aria-label="Documentation">`)
	for _, group := range GroupsInOrder(p.Guides) {
		entries := InGroup(p.Guides, group)
		if group == GroupRef && len(entries) == 0 {
			entries = []Guide{{URL: ReferenceURL, Title: "Every signal, source and setting"}}
		}
		if len(entries) == 0 {
			continue
		}
		b.WriteString(`<p class="sidehead">` + esc(group) + `</p><ul>`)
		for _, g := range entries {
			class := ""
			if g.URL == p.Current {
				class = ` class="on"`
			}
			// A fragment addresses this page; everything else is another page on the site, and
			// absolute so the file also works opened from disk.
			href := Host + g.URL
			if strings.HasPrefix(g.URL, "#") {
				href = g.URL
			}
			b.WriteString(`<li><a` + class + ` href="` + href + `">` + esc(g.Title) + `</a></li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</nav></aside>`)
}

// favicon is the mark the overview page uses, inline because a served page fetches nothing.
const favicon = `data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'%3E%3Crect width='32' height='32' rx='8' fill='%230f766e'/%3E%3Cpath d='M9 22 16 10l7 12' stroke='%23ffffff' stroke-width='2.6' fill='none' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E`

// breadcrumbs is the page's position under the overview as schema.org JSON-LD. json.Marshal
// escapes <, > and &, so a title cannot close the script element.
func breadcrumbs(p *pageSpec) string {
	type item struct {
		Type     string `json:"@type"`
		Position int    `json:"position"`
		Name     string `json:"name"`
		Item     string `json:"item"`
	}
	trail := []item{{"ListItem", 1, "assaio", Host + "/"}, {"ListItem", 2, "Docs", Host + "/docs"}}
	if p.URL != "/docs" {
		trail = append(trail, item{"ListItem", 3, p.Title, Host + p.URL})
	}
	out, err := json.Marshal(map[string]any{
		"@context":        "https://schema.org",
		"@type":           "BreadcrumbList",
		"itemListElement": trail,
	})
	if err != nil {
		return "{}"
	}
	return string(out)
}
