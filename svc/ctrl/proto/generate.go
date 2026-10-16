package proto

//go:generate go run github.com/unkeyed/unkey/tools/generate-proto-message -source ../../../pkg/mysql/types/deployment_capabilities.go -type DeploymentCapabilities -package ctrl.v1 -go-package github.com/unkeyed/unkey/gen/proto/ctrl/v1;ctrlv1 -out ./ctrl/v1/deployment_capabilities.proto
//go:generate go tool buf generate --template ./buf.gen.yaml --path ./ctrl
//go:generate go tool buf generate --template ./buf.gen.restate.yaml --path ./hydra
//go:generate go tool buf generate --template ./buf.gen.ts.yaml --path ./ctrl
//go:generate go run github.com/unkeyed/unkey/tools/generate-rpc-clients -source ../../../gen/proto/ctrl/v1/ctrlv1connect/*.connect.go -out ../../../gen/rpc/ctrl/
