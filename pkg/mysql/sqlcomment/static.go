package sqlcomment

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/buildinfo"
)

const applicationName = "unkey"

// Static tags are fixed for the lifetime of a database connection pool.
type Static struct {
	Application string
	Service     string
	Region      string
	ReleaseSHA  string

	prefix string
}

// Enabled reports whether queries should be annotated. Annotation is off when
// Service is empty so tests and local tools that omit tags stay unchanged.
func (s Static) Enabled() bool {
	return s.Service != ""
}

func (s Static) staticPrefix() string {
	return s.prefix
}

// Disabled returns static tags with annotation turned off. Use in tests, dev
// CLIs, and integration harnesses that do not need Query Insights attribution.
func Disabled() Static {
	return Static{
		Application: "",
		Service:     "",
		Region:      "",
		ReleaseSHA:  "",
		prefix:      "",
	}
}

// ForService builds the standard static tag set for a Unkey service process.
// ReleaseSHA comes from [buildinfo.Revision], which Bazel and goreleaser inject
// at link time. We do not tag a deployment environment name: infra does not set
// one on platform pods (cluster labels like production001/canary stay outside
// the process).
func ForService(service, region string) Static {
	s := Static{
		Application: applicationName,
		Service:     service,
		Region:      region,
		ReleaseSHA:  shortRevision(buildinfo.Revision),
		prefix:      "",
	}
	s.prefix = buildStaticPrefix(s)
	return s
}

func buildStaticPrefix(s Static) string {
	var b strings.Builder
	b.Grow(128)
	appendTag(&b, "application", s.Application)
	appendTag(&b, "service", s.Service)
	appendTag(&b, "region", s.Region)
	appendTag(&b, "release_sha", s.ReleaseSHA)
	return b.String()
}

func shortRevision(revision string) string {
	if revision == "" || revision == "unknown" {
		return ""
	}
	if len(revision) <= 7 {
		return revision
	}
	return revision[:7]
}
