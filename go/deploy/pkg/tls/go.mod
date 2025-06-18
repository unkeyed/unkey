module github.com/unkeyed/unkey/go/deploy/pkg/tls

go 1.21

require (
	github.com/unkeyed/unkey/go/deploy/pkg/spiffe v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.62.0
)

replace github.com/unkeyed/unkey/go/deploy/pkg/spiffe => ../spiffe

require (
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/go-jose/go-jose/v3 v3.0.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/spiffe/go-spiffe/v2 v2.1.6 // indirect
	github.com/zeebo/errs v1.3.0 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
)

