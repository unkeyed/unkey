package middleware

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/vault/storage"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type tracingMiddleware struct {
	name string
	next storage.Storage
}

func WithTracing(name string, next storage.Storage) storage.Storage {
	return &tracingMiddleware{
		name: name,
		next: next,
	}
}

func (tm *tracingMiddleware) PutObject(ctx context.Context, key string, object []byte) error {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("storage.%s.PutObject", tm.name))
	defer span.End()
	span.SetAttributes(attribute.String("key", key))
	err := tm.next.PutObject(ctx, key, object)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (tm *tracingMiddleware) GetObject(ctx context.Context, key string) ([]byte, bool, error) {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("storage.%s.GetObject", tm.name))
	defer span.End()
	span.SetAttributes(attribute.String("key", key))
	object, found, err := tm.next.GetObject(ctx, key)
	span.SetAttributes(attribute.Bool("found", found))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return object, found, err
}

func (tm *tracingMiddleware) ListObjectKeys(ctx context.Context, prefix string) ([]string, error) {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("storage.%s.ListObjectKeys", tm.name))
	defer span.End()
	span.SetAttributes(attribute.String("prefix", prefix))
	keys, err := tm.next.ListObjectKeys(ctx, prefix)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return keys, err
}

func (tm *tracingMiddleware) Key(shard string, dekID string) string {
	return tm.next.Key(shard, dekID)
}

func (tm *tracingMiddleware) Latest(shard string) string {
	return tm.next.Latest(shard)
}
