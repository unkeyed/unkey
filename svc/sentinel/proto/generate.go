package proto

//go:generate go tool buf generate --template ./buf.gen.yaml --path ./policies
//go:generate go tool buf generate --template ./buf.gen.yaml --path ./config
//go:generate go tool buf generate --template ./buf.gen.ts.yaml --path ./policies
//go:generate go tool buf generate --template ./buf.gen.ts.yaml --path ./config
