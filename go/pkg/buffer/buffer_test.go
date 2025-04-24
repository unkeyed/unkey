// buffer_test.go

package buffer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []Config{
		{
			Name:     "default_settings",
			Capacity: 10_000,
			Drop:     false,
		},
		{
			Name:     "custom_capacity",
			Capacity: 5000,
			Drop:     false,
		},
		{
			Name:     "with_drop_enabled",
			Capacity: 10_000,
			Drop:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			b := New[int](tt)

			assert.Equal(t, tt.Capacity, cap(b.c), "channel capacity should match")
			assert.Equal(t, tt.Drop, b.drop, "drop behavior should match")
		})
	}
}

func TestBuffer(t *testing.T) {
	tests := []struct {
		config  Config
		name    string
		input   []int
		wantLen int
	}{
		{
			config: Config{
				Name:     "a",
				Capacity: 5,
				Drop:     false,
			},
			name:    "Should buffer all elements when under capacity",
			input:   []int{1, 2, 3},
			wantLen: 3,
		},
		{
			config: Config{
				Name:     "b",
				Capacity: 3,
				Drop:     true,
			},
			name:    "Should drop elements when buffer is full and drop is enabled",
			input:   []int{1, 2, 3, 4, 5},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New[int](tt.config)

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
		b := New[int](Config{
			Name:     "a",
			Capacity: 2,
			Drop:     false,
		})

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
		b := New[CustomEvent](Config{
			Name:     "custom_event",
			Capacity: 1000,
			Drop:     false,
		})
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
