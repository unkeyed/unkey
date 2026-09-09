package proto

//go:generate go tool -modfile=../tools/buf/go.mod buf generate
//go:generate go tool -modfile=../tools/buf/go.mod buf generate --template ./buf.gen.ts.yaml --path ./logdrain
