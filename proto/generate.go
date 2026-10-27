package proto

//go:generate go tool buf generate
//go:generate go tool buf generate --template ./buf.gen.ts.yaml --path ./logdrain --path ./notification
//go:generate go tool buf generate --template ./buf.gen.notification.ts.yaml --path ./notification
