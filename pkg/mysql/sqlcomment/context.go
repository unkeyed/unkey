package sqlcomment

import "context"

type dynamicKey struct{}

// Dynamic tags vary per request or background job.
type Dynamic struct {
	Route  string
	Source string
}

// WithDynamic attaches per-request tags to ctx for the mysql replica wrapper.
func WithDynamic(ctx context.Context, tags Dynamic) context.Context {
	if tags.Route == "" && tags.Source == "" {
		return ctx
	}
	return context.WithValue(ctx, dynamicKey{}, tags)
}

// DynamicFromContext returns tags previously stored with [WithDynamic].
func DynamicFromContext(ctx context.Context) Dynamic {
	if ctx == nil {
		return Dynamic{Route: "", Source: ""}
	}
	tags, ok := ctx.Value(dynamicKey{}).(Dynamic)
	if !ok {
		return Dynamic{Route: "", Source: ""}
	}
	return tags
}
