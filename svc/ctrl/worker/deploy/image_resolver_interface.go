package deploy

import "context"

// ImageResolver resolves a mutable OCI image reference to an immutable digest.
type ImageResolver interface {
	Resolve(ctx context.Context, imageReference string) (string, error)
}
