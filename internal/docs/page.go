package docs

import "strings"

// Host prefixes every link a generated page emits. Absolute rather than host-relative so the
// same file reads correctly opened straight from disk -- the posture the offline dashboard has,
// applied to the pages that document it.
const Host = "https://assaio.dev"

// pageSpec is one published page. Description feeds the meta tags a shared link renders from;
// Current marks which sidebar entry the reader is on.
type pageSpec struct {
	Title       string
	Description string
	URL         string
	Current     string
	Body        string
	Guides      []Guide
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
<meta property="og:image:alt" content="assaio — measure whether AI coding spend is delivering">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:image" content="https://assaio.dev/og.png">
<link rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'%3E%3Crect width='32' height='32' rx='7' fill='%233E8B79'/%3E%3Cpath d='M9 22 16 10l7 12' stroke='%23EFEAE1' stroke-width='2.4' fill='none' stroke-linejoin='round'/%3E%3C/svg%3E">
<style>` + pageStyle + `</style>
</head>
<body><div class="wrap">
<header class="top">
  <a class="brand" href="` + Host + `/">assaio</a>
  <nav class="topnav">
    <a href="` + Host + `/">Overview</a>
    <a href="` + Host + `/docs">Docs</a>
    <a href="` + Host + ReferenceURL + `">Reference</a>
    <a href="https://github.com/assaio/assaio">GitHub</a>
    <a href="https://github.com/assaio/assaio/releases">Releases</a>
  </nav>
</header>
<div class="layout">`)
	writeSidebar(&b, p)
	b.WriteString(`<main class="doc">` + p.Body + `</main>
</div>
<footer>
<p>Generated from the repository: every page here is rendered from the Markdown it is written in,
or from the binary's own registries, and a check fails the build when the two disagree. This page
names no release — which version is current is stated once, on the
<a href="https://github.com/assaio/assaio/releases">releases page</a>, where it cannot fall behind.
It loads nothing: no fonts, no analytics, no third-party requests.</p>
</footer>
</div>` + themeScript + `</body></html>
`)
	return []byte(b.String())
}

func writeSidebar(b *strings.Builder, p *pageSpec) {
	b.WriteString(`<aside class="side"><nav>`)
	for _, group := range GroupsInOrder() {
		entries := InGroup(p.Guides, group)
		if group == GroupRef {
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
			b.WriteString(`<li><a` + class + ` href="` + Host + g.URL + `">` + esc(g.Title) + `</a></li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</nav></aside>`)
}

// themeScript adopts the choice the main page stored. Without it a reader who set dark there
// lands on a light page, because the pages share a palette but not a document. There is no
// toggle here on purpose: one control, on the page that has one.
const themeScript = `<script>
try{var t=localStorage.getItem('assaio-theme'); if(t) document.documentElement.setAttribute('data-theme',t);}catch(e){}
</script>`
