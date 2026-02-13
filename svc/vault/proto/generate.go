package proto

//go:generate go tool buf generate
//go:generate ../../../dist/generate-rpc-clients -source ../../../gen/proto/vault/v1/vaultv1connect/*.connect.go -out ../../../pkg/rpc/vault/
