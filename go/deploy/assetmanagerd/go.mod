module github.com/unkeyed/unkey/go/deploy/assetmanagerd

go 1.25.1

require (
	connectrpc.com/connect v1.18.1
	github.com/caarlos0/env/v11 v11.3.1
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/oklog/ulid/v2 v2.1.1
	github.com/unkeyed/unkey/go v0.0.0-20250907150353-7f609cd7c284
	github.com/unkeyed/unkey/go/deploy/pkg/health v0.0.0-20250907150353-7f609cd7c284
	github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors v0.0.0-20250907150353-7f609cd7c284
	github.com/unkeyed/unkey/go/deploy/pkg/tls v0.0.0-20250907150353-7f609cd7c284
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.38.0
	go.opentelemetry.io/otel/exporters/prometheus v0.60.0
	go.opentelemetry.io/otel/metric v1.38.0
	go.opentelemetry.io/otel/sdk v1.38.0
	go.opentelemetry.io/otel/sdk/metric v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
	golang.org/x/net v0.43.0
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grafana/regexp v0.0.0-20250905093917-f7b3be9d1853 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/otlptranslator v0.0.2 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/unkeyed/unkey/go/deploy/pkg/spiffe v0.0.0-20250907150353-7f609cd7c284 // indirect
	github.com/unkeyed/unkey/go/deploy/pkg/tracing v0.0.0-20250907150353-7f609cd7c284 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.8.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250826171959-ef028d996bc1 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250826171959-ef028d996bc1 // indirect
	google.golang.org/grpc v1.75.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

replace github.com/unkeyed/unkey/go/deploy/pkg/tls => ../pkg/tls

replace github.com/unkeyed/unkey/go/deploy/pkg/spiffe => ../pkg/spiffe

replace github.com/unkeyed/unkey/go/deploy/pkg/health => ../pkg/health

replace github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors => ../pkg/observability/interceptors

replace github.com/unkeyed/unkey/go/deploy/pkg/tracing => ../pkg/tracing

replace github.com/unkeyed/unkey/go/deploy/builderd => ../builderd

replace github.com/unkeyed/unkey/go => ../../
