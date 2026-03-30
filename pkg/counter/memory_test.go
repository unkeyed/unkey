package counter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMemoryCounter_IncrementAndGet(t *testing.T) {
	c := NewMemory()
	ctx := context.Background()

	val, err := c.Increment(ctx, "key1", 5)
	require.NoError(t, err)
	require.Equal(t, int64(5), val)

	val, err = c.Increment(ctx, "key1", 3)
	require.NoError(t, err)
	require.Equal(t, int64(8), val)

	got, err := c.Get(ctx, "key1")
	require.NoError(t, err)
	require.Equal(t, int64(8), got)
}

func TestMemoryCounter_GetNonExistent(t *testing.T) {
	c := NewMemory()

	got, err := c.Get(context.Background(), "missing")
	require.NoError(t, err)
	require.Equal(t, int64(0), got)
}

func TestMemoryCounter_MultiGet(t *testing.T) {
	c := NewMemory()
	ctx := context.Background()

	_, _ = c.Increment(ctx, "a", 1)
	_, _ = c.Increment(ctx, "b", 2)

	result, err := c.MultiGet(ctx, []string{"a", "b", "missing"})
	require.NoError(t, err)
	require.Equal(t, int64(1), result["a"])
	require.Equal(t, int64(2), result["b"])
	require.Equal(t, int64(0), result["missing"])
}

func TestMemoryCounter_Decrement(t *testing.T) {
	c := NewMemory()
	ctx := context.Background()

	_, _ = c.Increment(ctx, "key1", 10)

	val, err := c.Decrement(ctx, "key1", 3)
	require.NoError(t, err)
	require.Equal(t, int64(7), val)
}

func TestMemoryCounter_DecrementIfExists(t *testing.T) {
	c := NewMemory()
	ctx := context.Background()

	// Key doesn't exist
	val, existed, success, err := c.DecrementIfExists(ctx, "key1", 5)
	require.NoError(t, err)
	require.Equal(t, int64(0), val)
	require.False(t, existed)
	require.False(t, success)

	// Initialize key
	_, _ = c.Increment(ctx, "key1", 10)

	// Sufficient credits
	val, existed, success, err = c.DecrementIfExists(ctx, "key1", 3)
	require.NoError(t, err)
	require.Equal(t, int64(7), val)
	require.True(t, existed)
	require.True(t, success)

	// Insufficient credits
	val, existed, success, err = c.DecrementIfExists(ctx, "key1", 100)
	require.NoError(t, err)
	require.Equal(t, int64(7), val)
	require.True(t, existed)
	require.False(t, success)
}

func TestMemoryCounter_SetIfNotExists(t *testing.T) {
	c := NewMemory()
	ctx := context.Background()

	set, err := c.SetIfNotExists(ctx, "key1", 42)
	require.NoError(t, err)
	require.True(t, set)

	set, err = c.SetIfNotExists(ctx, "key1", 99)
	require.NoError(t, err)
	require.False(t, set)

	got, err := c.Get(ctx, "key1")
	require.NoError(t, err)
	require.Equal(t, int64(42), got)
}

func TestMemoryCounter_Delete(t *testing.T) {
	c := NewMemory()
	ctx := context.Background()

	_, _ = c.Increment(ctx, "key1", 10)
	require.NoError(t, c.Delete(ctx, "key1"))

	got, err := c.Get(ctx, "key1")
	require.NoError(t, err)
	require.Equal(t, int64(0), got)
}

func TestMemoryCounter_TTLExpiry(t *testing.T) {
	c := NewMemory().(*memoryCounter)
	ctx := context.Background()

	// Set a key with TTL that's already expired
	c.mu.Lock()
	c.entries["expired"] = memoryEntry{
		value:  100,
		expiry: time.Now().Add(-1 * time.Second),
	}
	c.mu.Unlock()

	got, err := c.Get(ctx, "expired")
	require.NoError(t, err)
	require.Equal(t, int64(0), got)

	// SetIfNotExists should succeed on expired key
	set, err := c.SetIfNotExists(ctx, "expired", 50)
	require.NoError(t, err)
	require.True(t, set)
}

func TestMemoryCounter_Close(t *testing.T) {
	c := NewMemory()
	require.NoError(t, c.Close())
}
