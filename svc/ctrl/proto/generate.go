package proto

//go:generate go tool buf generate --template ./buf.gen.yaml --path ./ctrl
//go:generate go tool buf generate --template ./buf.gen.restate.yaml --path ./hydra
//go:generate go tool buf generate --template ./buf.gen.ts.yaml --path ./ctrl
//go:generate ../../../dist/generate-rpc-clients -source ../../../gen/proto/ctrl/v1/ctrlv1connect/*.connect.go -out ../../../pkg/rpc/ctrl/
