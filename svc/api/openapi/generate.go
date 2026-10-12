package openapi

//go:generate go run generate_bundle.go -input openapi-split.yaml -output openapi-generated.yaml
//go:generate go tool -modfile=../../../tools/oapi-codegen/go.mod oapi-codegen -config=config.yaml ./openapi-generated.yaml
