package openapi

import (
	_ "embed"
)

const (
	BearerScopes        = "bearer.Scopes"
	PortalSessionScopes = "portalSession.Scopes"
)

// Spec is the OpenAPI specification for the service
// It's loaded from our openapi file and embedded into the binary
//
//go:embed openapi-generated.yaml
var Spec []byte
