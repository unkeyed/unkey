package openapi

//go:generate go run generate_bundle.go -input openapi-split.yaml -output openapi-generated.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config=config.yaml ./openapi-generated.yaml
//go:generate go tool github.com/mailru/easyjson/easyjson -all gen.go
