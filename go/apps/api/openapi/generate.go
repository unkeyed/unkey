package openapi

//go:generate go run generate_bundle.go -input openapi-split.yaml -output openapi-bundled.yaml
//go:generate go tool oapi-codegen -config=config.yaml ./openapi.yaml
