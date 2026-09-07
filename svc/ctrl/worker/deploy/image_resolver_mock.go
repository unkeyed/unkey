package deploy

import "context"

// ImageResolverFunc adapts a function for use as an [ImageResolver].
type ImageResolverFunc func(ctx context.Context, imageReference string) (string, error)

// Resolve calls f.
func (f ImageResolverFunc) Resolve(ctx context.Context, imageReference string) (string, error) {
	return f(ctx, imageReference)
}
