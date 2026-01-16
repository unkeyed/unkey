// Package exhaustruct provides an exhaustruct analyzer for use with nogo.
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
