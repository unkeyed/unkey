// Package errorpage renders HTML error pages for frontline.
//
// Frontline shows error pages for routing/proxy failures and for policy
// rejections (auth, rate limits, firewall). The [Renderer] interface allows
// swapping the template, e.g. for custom domains with branded error pages.
//
// # Template
//
// The default implementation embeds error.go.tmpl at compile time and
// renders it with [html/template]. The template receives a [Data] struct
// and supports dark/light mode via prefers-color-scheme.
//
// # Content Negotiation
//
// This package only produces HTML. The caller (middleware or proxy) is
// responsible for checking the Accept header and falling back to JSON
// when the client prefers it.
package errorpage
