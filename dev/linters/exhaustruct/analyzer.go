// Package exhaustruct provides an exhaustruct analyzer for use with nogo.
//
// Exhaustruct requires all struct fields to be explicitly initialized when
// creating a struct literal. This prevents bugs where new fields are added
// to a struct but existing call sites silently use zero values instead of
// providing meaningful defaults.
//
// This wrapper skips test files (_test.go) because test fixtures often use
// partial initialization intentionally. It also excludes a curated list of
// external types (protobuf, standard library, AWS SDK, etc.) where exhaustive
// initialization would be impractical or impossible.
//
// # Why This Matters
//
// When you add a required Timeout field to a Config struct, every place that
// creates a Config should specify a timeout. Without exhaustruct, those call
// sites silently use 0 (meaning "no timeout" or "infinite"). The linter forces
// explicit decisions about every field, catching these issues at build time.
//
// # Excluded Types
//
// The [excludePatterns] list exempts types where exhaustive initialization is
// unreasonable: protobuf messages with dozens of internal fields, http.Request
// with optional fields, test containers, and similar cases.
package exhaustruct

import (
	"golang.org/x/tools/go/analysis"

	"dev.gaijin.team/go/exhaustruct/v4/analyzer"

	"github.com/unkeyed/unkey/dev/linters/nolint"
)

// Analyzer is the exhaustruct analyzer wrapped with nolint support.
var Analyzer *analysis.Analyzer = mustCreateAnalyzer()

// excludePatterns contains regex patterns for types to exclude from exhaustruct checks.
var excludePatterns = []string{
	// Protobuf generated types
	`^google\.golang\.org/protobuf/internal/filedesc\..*$`,
	`^google\.golang\.org/protobuf/internal/filetype\..*$`,
	`^google\.golang\.org/protobuf/reflect/protoreflect\..*$`,
	`^github\.com/unkeyed/unkey/gen/proto/.*$`,

	// Standard library
	`^net/http\.Client$`,
	`^net/http\.Cookie$`,
	`^net/http\.Request$`,
	`^net/http\.Response$`,
	`^net/http\.Server$`,
	`^net/http\.Transport$`,
	`^net/url\.URL$`,
	`^os/exec\.Cmd$`,
	`^reflect\.StructField$`,
	`^database/sql\.Null.*$`,

	// Prometheus
	`^github\.com/prometheus/client_golang/.+Opts$`,

	// Testcontainers
	`^github\.com/testcontainers/testcontainers-go.+Request$`,
	`^github\.com/testcontainers/testcontainers-go\.FromDockerfile$`,
	`^github\.com/ory/dockertest/v3.*$`,

	// Go tools
	`^golang\.org/x/tools/go/analysis\.Analyzer$`,

	// Protobuf
	`^google\.golang\.org/protobuf/.+Options$`,

	// YAML
	`^gopkg\.in/yaml\.v3\.Node$`,

	// CLI frameworks
	`^github\.com/urfave/cli/v3.*$`,

	// ClickHouse
	`^github\.com/ClickHouse/clickhouse-go/v2.*$`,
	`^github\.com/AfterShip/clickhouse-sql-parser/parser.*$`,

	// AWS SDK
	`^github\.com/aws/aws-sdk-go-v2/.*$`,

	// Unkey internal
	`^github\.com/unkeyed/unkey/svc/api/openapi\.Meta$`,
	`^github\.com/unkeyed/unkey/pkg/cli\.Command$`,

	// ORM
	`^gorm\.io/gorm.*$`,

	// Redis
	`^github\.com/redis/go-redis/v9\.Options$`,
	`^github\.com/go-redis/redis/v8\.Options$`,

	// Kubernetes
	`^k8s\.io/api/core/v1.*$`,
	`^k8s\.io/api/apps/v1.*$`,
	`^k8s\.io/apimachinery/pkg/apis/meta/v1.*$`,
	`^sigs\.k8s\.io/controller-runtime/pkg/client.*$`,

	// Docker/Moby
	`^github\.com/moby/buildkit/client.*$`,
	`^github\.com/docker/docker/api/types/container.*$`,
	`^github\.com/docker/docker/api/types/image.*$`,
	`^github\.com/docker/docker/api/types/registry\.AuthConfig$`,
	`^github\.com/docker/cli/cli/config/configfile\.ConfigFile$`,
}

func mustCreateAnalyzer() *analysis.Analyzer {
	a, err := analyzer.NewAnalyzer(analyzer.Config{
		ExcludeRx: excludePatterns,
	})
	if err != nil {
		panic("exhaustruct: failed to create analyzer: " + err.Error())
	}

	return nolint.WrapSkipPatterns(a, "_test.go")
}
