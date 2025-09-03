module github.com/unkeyed/unkey/go/deploy/metald/client

go 1.25.0

require (
	connectrpc.com/connect v1.18.1
	github.com/unkeyed/unkey/go v0.0.0-00010101000000-000000000000
	github.com/unkeyed/unkey/go/deploy/pkg/tls v0.0.0
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/unkeyed/unkey/go/deploy/pkg/spiffe v0.0.0-00010101000000-000000000000 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

replace github.com/unkeyed/unkey/go/deploy/metald => ..

replace github.com/unkeyed/unkey/go/deploy/pkg/tls => ../../pkg/tls

replace github.com/unkeyed/unkey/go/deploy/pkg/spiffe => ../../pkg/spiffe

replace github.com/unkeyed/unkey/go => ../../..
