package proto

//go:generate go tool buf generate --template ./buf.gen.yaml --path ./ctrl
//go:generate go tool buf generate --template ./buf.gen.restate.yaml --path ./hydra
//go:generate go tool buf generate --template ./buf.gen.ts.yaml --path ./ctrl
