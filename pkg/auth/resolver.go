package auth

import (
	"context"

	"github.com/unkeyed/unkey/pkg/zen"
)

// Resolver turns one credential source into a principal or yields to the next resolver.
//
// Implementations must return (nil, nil) when the request does not contain a
// credential they understand. They must return an error when the request does
// contain their credential type but verification fails. A non-nil principal
// means the resolver fully authenticated the request, populated WorkspaceID,
// Subject, Type, Source, and Permissions, and no later resolver will run.
type Resolver interface {
	Resolve(ctx context.Context, sess *zen.Session) (*Principal, error)
}
