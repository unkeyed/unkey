package proto

//go:generate go tool buf generate
//go:generate go run github.com/unkeyed/unkey/tools/generate-rpc-clients -source ../../../gen/proto/krane/v1/kranev1connect/*.connect.go -out ../../../gen/rpc/krane/
