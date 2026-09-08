package proto

//go:generate go tool -modfile=../../../tools/buf/go.mod buf generate
//go:generate go tool -modfile=../../../tools/buf/go.mod buf generate --template ./buf.gen.ts.yaml --path ./vault/v1/service.proto
//go:generate go run github.com/unkeyed/unkey/tools/generate-rpc-clients -source ../../../gen/proto/vault/v1/vaultv1connect/*.connect.go -out ../../../gen/rpc/vault/
