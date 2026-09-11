package openapi

// Keep SDK schema names stable independently of runtime validator upgrades.
//go:generate go run -modfile=bundle.mod generate_bundle.go -input openapi-split.yaml -output openapi-generated.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config=config.yaml ./openapi-generated.yaml
