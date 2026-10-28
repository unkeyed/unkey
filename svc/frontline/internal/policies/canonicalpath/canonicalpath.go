// Package canonicalpath reduces a request path to the form app frameworks
// route on, so gateway policies and per-path rate limits cannot be bypassed
// with path variants such as "/admin/", "/admin;x", or "/x/../admin".
//
// It lives outside package policies because the policy engine imports the
// ratelimit package, which also needs the canonical path.
package canonicalpath

import (
	"path"
	"strings"
)

// Clean strips ";" parameters from every segment, then collapses duplicate
// slashes, resolves "." and ".." segments, and drops any trailing slash. Case
// is preserved. Parameters are stripped first because Tomcat resolves
// "/public/..;/admin" to "/admin".
func Clean(p string) string {
	segments := strings.Split(p, "/")
	for i, segment := range segments {
		segments[i], _, _ = strings.Cut(segment, ";")
	}
	return path.Clean("/" + strings.Join(segments, "/"))
}

// Candidates returns the paths a policy path expression is tested against: the
// cleaned decoded path and, when it differs, the cleaned escaped path, each
// with and without a trailing slash. The escaped form matters because the
// upstream receives it, and frameworks such as Express do not treat "%2F" as a
// separator. The raw path is never a candidate, so an earlier policy cannot
// claim a request through a variant the app routes elsewhere.
func Candidates(decoded, escaped string) []string {
	cleaned := Clean(decoded)
	candidates := withTrailingSlash(make([]string, 0, 4), cleaned)
	if escapedCleaned := Clean(escaped); escapedCleaned != cleaned {
		candidates = withTrailingSlash(candidates, escapedCleaned)
	}
	return candidates
}

func withTrailingSlash(candidates []string, p string) []string {
	if p == "/" {
		return append(candidates, p)
	}
	return append(candidates, p, p+"/")
}
