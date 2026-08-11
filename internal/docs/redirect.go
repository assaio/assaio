package docs

import "strings"

// RedirectFile is the page left behind where the reference used to live. Workers Assets serves
// static files and cannot redirect, and the URL was published and linked before the reference
// moved inside the documentation -- so a page that says where it went is the only thing that
// keeps a shared link working.
const RedirectFile = "site/reference.html"

// Redirect renders that page: a meta refresh for a browser, and a sentence for anyone whose
// browser ignores it. It fetches nothing, which is what lets it live under site/ at all.
func Redirect() []byte {
	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="refresh" content="0; url=` + ReferenceURL + `">
<title>The reference moved — assaio</title>
<meta name="description" content="The assaio reference now lives inside the documentation, at /docs/reference.">
<link rel="canonical" href="` + Host + ReferenceURL + `">
<meta property="og:type" content="website">
<meta property="og:url" content="` + Host + ReferenceURL + `">
<meta property="og:title" content="assaio reference">
<meta property="og:description" content="Every signal, source, validator, command and setting, generated from the binary.">
<meta property="og:image" content="` + Host + `/og.png">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:image:alt" content="assaio — measure whether AI coding spend is delivering">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:image" content="` + Host + `/og.png">
<style>` + baseStyle + `</style>
</head>
<body><div class="wrap"><header class="top"><a class="brand" href="` + Host + `/">assaio</a></header>
<p style="padding:2.5rem 0">The reference moved inside the documentation. It is now at
<a href="` + Host + ReferenceURL + `">` + Host + ReferenceURL + `</a>.</p>
</div></body></html>
`)
	return []byte(b.String())
}
