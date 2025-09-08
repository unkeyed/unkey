module github.com/unkeyed/unkey/go/deploy/metald

go 1.25

require (
	connectrpc.com/connect v1.18.1
	github.com/firecracker-microvm/firecracker-go-sdk v1.0.0
	github.com/mattn/go-sqlite3 v1.14.28
	github.com/prometheus/client_golang v1.23.1
	github.com/stretchr/testify v1.11.1
	github.com/unkeyed/unkey/go v0.0.0-00010101000000-000000000000
	github.com/unkeyed/unkey/go/deploy/pkg/health v0.0.0-00010101000000-000000000000
	github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors v0.0.0-00010101000000-000000000000
	github.com/unkeyed/unkey/go/deploy/pkg/tls v0.0.0-00010101000000-000000000000
	github.com/vishvananda/netlink v1.3.1
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.38.0
	go.opentelemetry.io/otel/exporters/prometheus v0.59.0
	go.opentelemetry.io/otel/metric v1.38.0
	go.opentelemetry.io/otel/sdk v1.38.0
	go.opentelemetry.io/otel/sdk/metric v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
	golang.org/x/net v0.43.0
	golang.org/x/sys v0.35.0
	google.golang.org/protobuf v1.36.8
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/fifo v1.1.0 // indirect
	github.com/containernetworking/cni v1.3.0 // indirect
	github.com/containernetworking/plugins v1.7.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.1 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/runtime v0.28.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grafana/regexp v0.0.0-20240518133315-a468a5bfb3bc // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/unkeyed/unkey/go/deploy/pkg/spiffe v0.0.0-00010101000000-000000000000 // indirect
	github.com/unkeyed/unkey/go/deploy/pkg/tracing v0.0.0-00010101000000-000000000000 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	go.mongodb.org/mongo-driver v1.17.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/proto/otlp v1.8.0 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250826171959-ef028d996bc1 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250826171959-ef028d996bc1 // indirect
	google.golang.org/grpc v1.75.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/unkeyed/unkey/go/deploy/billaged => ../billaged

replace github.com/unkeyed/unkey/go/deploy/builderd => ../builderd

replace github.com/unkeyed/unkey/go/deploy/assetmanagerd => ../assetmanagerd

replace github.com/unkeyed/unkey/go/deploy/pkg/tls => ../pkg/tls

replace github.com/unkeyed/unkey/go/deploy/pkg/spiffe => ../pkg/spiffe

replace github.com/unkeyed/unkey/go/deploy/pkg/health => ../pkg/health

replace github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors => ../pkg/observability/interceptors

replace github.com/unkeyed/unkey/go/deploy/pkg/tracing => ../pkg/tracing

replace github.com/mitchellh/osext => github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0

replace github.com/unkeyed/unkey/go => ../../
