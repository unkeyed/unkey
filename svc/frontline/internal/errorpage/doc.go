// Package errorpage renders HTML error pages for frontline.
//
// Frontline shows error pages for its own errors (routing failures, proxy
// errors) and for sentinel errors (auth rejections, rate limits). The
// [Renderer] interface allows swapping the template, e.g. for custom
// domains with branded error pages.
//
// # Template
//
// The default implementation embeds error.go.tmpl at compile time and
// renders it with [html/template]. The template receives a [Data] struct
// and supports dark/light mode via prefers-color-scheme.
//
// # Content Negotiation
//
// This package only produces HTML. The caller (frontline middleware or
// proxy) is responsible for checking the Accept header and falling back
// to JSON when the client prefers it.
package errorpage
