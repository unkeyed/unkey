package wide

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventContext_Set(t *testing.T) {
	ev := NewEventContext(EventConfig{})

	ev.Set("key1", "value1")
	ev.Set("key2", 42)
	ev.Set("key3", true)

	val, ok := ev.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	val, ok = ev.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	val, ok = ev.Get("key3")
	assert.True(t, ok)
	assert.Equal(t, true, val)

	// Non-existent key
	val, ok = ev.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestEventContext_SetMany(t *testing.T) {
	ev := NewEventContext(EventConfig{})

	ev.SetMany(map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	})

	assert.Equal(t, 3, ev.FieldCount())

	val, ok := ev.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestEventContext_MarkError(t *testing.T) {
	ev := NewEventContext(EventConfig{})

	assert.False(t, ev.HasError())

	ev.MarkError()

	assert.True(t, ev.HasError())
}

func TestEventContext_Duration(t *testing.T) {
	ev := NewEventContext(EventConfig{})

	time.Sleep(10 * time.Millisecond)

	duration := ev.Duration()
	assert.GreaterOrEqual(t, duration.Milliseconds(), int64(10))

	durationMs := ev.DurationMs()
	assert.GreaterOrEqual(t, durationMs, int64(10))
}

func TestEventContext_Fields(t *testing.T) {
	ev := NewEventContext(EventConfig{})

	ev.Set("key1", "value1")
	ev.Set("key2", 42)

	fields := ev.Fields()

	// Fields returns key-value pairs as a flat slice
	assert.Equal(t, 4, len(fields))

	// Convert to map for easier assertion
	fieldsMap := make(map[string]any)
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		require.True(t, ok)
		fieldsMap[key] = fields[i+1]
	}

	assert.Equal(t, "value1", fieldsMap["key1"])
	assert.Equal(t, 42, fieldsMap["key2"])
}

func TestEventContext_FieldsMap(t *testing.T) {
	ev := NewEventContext(EventConfig{})

	ev.Set("key1", "value1")
	ev.Set("key2", 42)

	fieldsMap := ev.FieldsMap()

	assert.Equal(t, 2, len(fieldsMap))
	assert.Equal(t, "value1", fieldsMap["key1"])
	assert.Equal(t, 42, fieldsMap["key2"])
}

func TestEventContext_Concurrency(t *testing.T) {
	ev := NewEventContext(EventConfig{})

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ev.Set("key"+string(rune(i)), i)
		}(i)
	}

	wg.Wait()

	assert.Equal(t, numGoroutines, ev.FieldCount())
}

func TestEventContext_StartTime(t *testing.T) {
	before := time.Now()
	ev := NewEventContext(EventConfig{})
	after := time.Now()

	startTime := ev.StartTime()

	assert.True(t, !startTime.Before(before), "start time should be >= before")
	assert.True(t, !startTime.After(after), "start time should be <= after")
}
