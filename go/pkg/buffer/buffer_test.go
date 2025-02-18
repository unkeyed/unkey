// buffer_test.go

package buffer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		drop     bool
	}{
		{
			name:     "default_settings",
			capacity: 10_000,
			drop:     false,
		},
		{
			name:     "custom_capacity",
			capacity: 5000,
			drop:     false,
		},
		{
			name:     "with_drop_enabled",
			capacity: 10_000,
			drop:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New[int](tt.capacity, tt.drop)

			assert.Equal(t, tt.capacity, cap(b.c), "channel capacity should match")
			assert.Equal(t, tt.drop, b.drop, "drop behavior should match")
		})
	}
}

func TestBuffer(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		drop     bool
		input    []int
		wantLen  int
	}{
		{
			name:     "Should buffer all elements when under capacity",
			capacity: 5,
			drop:     false,
			input:    []int{1, 2, 3},
			wantLen:  3,
		},
		{
			name:     "Should drop elements when buffer is full and drop is enabled",
			capacity: 3,
			drop:     true,
			input:    []int{1, 2, 3, 4, 5},
			wantLen:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New[int](tt.capacity, tt.drop)

			// Buffer elements
			for _, v := range tt.input {
				b.Buffer(v)
			}

			assert.Equal(t, tt.wantLen, len(b.c), "buffer length should match expected")

			// Verify elements can be received
			received := make([]int, 0, tt.wantLen)
			timeout := time.After(100 * time.Millisecond)

		receiveLoop:
			for {
				select {
				case v := <-b.c:
					received = append(received, v)
				case <-timeout:
					break receiveLoop
				}
			}

			assert.Equal(t, tt.wantLen, len(received), "received elements count should match expected")
		})
	}
}

func TestBlockingBehavior(t *testing.T) {
	t.Run("blocks_when_full", func(t *testing.T) {
		b := New[int](2, false)

		// Fill the buffer
		b.Buffer(1)
		b.Buffer(2)

		// Try to add another element with timeout
		done := make(chan bool)
		go func() {
			b.Buffer(3)
			done <- true
		}()

		select {
		case <-done:
			t.Error("Buffer should block when full")
		case <-time.After(100 * time.Millisecond):
			// Expected behavior - operation blocked
		}
	})
}

func TestCustomTypes(t *testing.T) {
	type CustomEvent struct {
		ID   string
		Data string
	}

	t.Run("custom_type_buffering", func(t *testing.T) {
		b := New[CustomEvent](1000, false)
		event := CustomEvent{ID: "1", Data: "test"}

		b.Buffer(event)

		select {
		case received := <-b.c:
			assert.Equal(t, event, received, "received event should match buffered event")
		default:
			t.Error("Expected to receive buffered event")
		}
	})
}
