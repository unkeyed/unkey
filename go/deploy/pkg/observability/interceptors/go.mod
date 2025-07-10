module github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors

go 1.24.4

require (
	connectrpc.com/connect v1.18.1
	github.com/unkeyed/unkey/go/deploy/pkg/tracing v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/metric v1.37.0
	go.opentelemetry.io/otel/trace v1.37.0
)

require (
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/unkeyed/unkey/go/deploy/pkg/tracing => ../../tracing
