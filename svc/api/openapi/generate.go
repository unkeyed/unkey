package openapi

//go:generate go run generate_bundle.go -input openapi-split.yaml -output openapi-generated.yaml -output30 openapi-generated-3.0.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config=config.yaml ./openapi-generated-3.0.yaml
