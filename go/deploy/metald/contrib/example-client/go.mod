module github.com/unkeyed/unkey/go/deploy/metald/contrib/example-client

go 1.24.3

toolchain go1.24.4

require (
	connectrpc.com/connect v1.18.1
	github.com/unkeyed/unkey/go/deploy/metald v0.0.0
	github.com/unkeyed/unkey/go/deploy/pkg/tls v0.0.0
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/go-jose/go-jose/v4 v4.0.4 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/unkeyed/unkey/go/deploy/pkg/spiffe v0.0.0-00010101000000-000000000000 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250519155744-55703ea1f237 // indirect
	google.golang.org/grpc v1.72.1 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/unkeyed/unkey/go/deploy/metald => ../..

replace github.com/unkeyed/unkey/go/deploy/pkg/tls => ../../../pkg/tls

replace github.com/unkeyed/unkey/go/deploy/pkg/spiffe => ../../../pkg/spiffe
