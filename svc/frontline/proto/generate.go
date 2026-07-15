package proto

//go:generate go tool buf generate --template ./buf.gen.yaml --path ./frontline/policies
//go:generate go tool buf generate --template ./buf.gen.yaml --path ./frontline/config
//go:generate go tool buf generate --template ./buf.gen.ts.yaml --path ./frontline/policies
//go:generate go tool buf generate --template ./buf.gen.ts.yaml --path ./frontline/config
