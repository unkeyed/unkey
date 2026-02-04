package wide

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithEventContext(t *testing.T) {
	ctx := context.Background()

	newCtx, ev := WithEventContext(ctx, EventConfig{})

	assert.NotNil(t, ev)
	assert.NotEqual(t, ctx, newCtx)

	// Verify EventContext is stored in context
	retrievedEv := FromContext(newCtx)
	assert.Equal(t, ev, retrievedEv)
}

func TestFromContext_NoEventContext(t *testing.T) {
	ctx := context.Background()

	ev := FromContext(ctx)

	assert.Nil(t, ev)
}

func TestSet_WithEventContext(t *testing.T) {
	ctx, ev := WithEventContext(context.Background(), EventConfig{})

	Set(ctx, "test_key", "test_value")

	val, ok := ev.Get("test_key")
	assert.True(t, ok)
	assert.Equal(t, "test_value", val)
}

func TestSet_NoEventContext(t *testing.T) {
	ctx := context.Background()

	// Should not panic, just no-op
	Set(ctx, "test_key", "test_value")

	// Verify nothing happened
	ev := FromContext(ctx)
	assert.Nil(t, ev)
}

func TestSetMany_WithEventContext(t *testing.T) {
	ctx, ev := WithEventContext(context.Background(), EventConfig{})

	SetMany(ctx, map[string]any{
		"key1": "value1",
		"key2": 42,
	})

	val, ok := ev.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	val, ok = ev.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestSetMany_NoEventContext(t *testing.T) {
	ctx := context.Background()

	// Should not panic, just no-op
	SetMany(ctx, map[string]any{
		"key1": "value1",
	})
}

func TestMarkError_WithEventContext(t *testing.T) {
	ctx, ev := WithEventContext(context.Background(), EventConfig{})

	assert.False(t, ev.HasError())

	MarkError(ctx)

	assert.True(t, ev.HasError())
}

func TestMarkError_NoEventContext(t *testing.T) {
	ctx := context.Background()

	// Should not panic, just no-op
	MarkError(ctx)
}

func TestGet_WithEventContext(t *testing.T) {
	ctx, ev := WithEventContext(context.Background(), EventConfig{})
	ev.Set("existing", "value")

	val, ok := Get(ctx, "existing")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	val, ok = Get(ctx, "nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestGet_NoEventContext(t *testing.T) {
	ctx := context.Background()

	val, ok := Get(ctx, "any_key")
	assert.False(t, ok)
	assert.Nil(t, val)
}
