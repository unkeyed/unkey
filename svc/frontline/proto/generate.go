package proto

//go:generate go tool -modfile=../../../tools/buf/go.mod buf generate --template ./buf.gen.yaml --path ./frontline/policies
//go:generate go tool -modfile=../../../tools/buf/go.mod buf generate --template ./buf.gen.yaml --path ./frontline/config
//go:generate go tool -modfile=../../../tools/buf/go.mod buf generate --template ./buf.gen.ts.yaml --path ./frontline/policies
//go:generate go tool -modfile=../../../tools/buf/go.mod buf generate --template ./buf.gen.ts.yaml --path ./frontline/config
