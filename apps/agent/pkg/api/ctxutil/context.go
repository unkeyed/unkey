package ctxutil

import "context"

const (
	request_id string = "request_id"
)

// getValue returns the value for the given key from the context or its zero value if it doesn't exist.
func getValue[T any](ctx context.Context, key string) T {
	val, ok := ctx.Value(key).(T)
	if !ok {
		var t T
		return t
	}
	return val
}

func GetRequestId(ctx context.Context) string {
	return getValue[string](ctx, request_id)
}

func SetRequestId(ctx context.Context, requestId string) context.Context {
	return context.WithValue(ctx, request_id, requestId)
}
