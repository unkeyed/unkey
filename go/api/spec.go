package api

import (
	_ "embed"
)

// Spec is the OpenAPI specification for the service
// It's loaded from our openapi file and embedded into the binary
//
//go:embed openapi.json
var Spec []byte
