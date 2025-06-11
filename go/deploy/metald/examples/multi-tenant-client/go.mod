module multi-tenant-client

go 1.24.3

replace metald => ../../

require (
	connectrpc.com/connect v1.18.1
	go.opentelemetry.io/otel v1.36.0
	metald v0.0.0-00010101000000-000000000000
)

require google.golang.org/protobuf v1.36.6 // indirect
