package conc

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestForEach_ProcessesAllItems(t *testing.T) {
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}

	var count atomic.Int64
	ForEach(context.Background(), items, func(_ context.Context, item *int) {
		count.Add(1)
	})

	if got := count.Load(); got != 100 {
		t.Errorf("expected 100 items processed, got %d", got)
	}
}

func TestForEach_PassesPointers(t *testing.T) {
	items := []int{1, 2, 3}

	ForEach(context.Background(), items, func(_ context.Context, item *int) {
		*item *= 10
	})

	for i, want := range []int{10, 20, 30} {
		if items[i] != want {
			t.Errorf("items[%d] = %d, want %d", i, items[i], want)
		}
	}
}

func TestForEach_EmptySlice(t *testing.T) {
	var items []int
	ForEach(context.Background(), items, func(_ context.Context, item *int) {
		t.Fatal("should not be called")
	})
}

func TestForEach_BoundsConcurrency(t *testing.T) {
	items := make([]int, 50)
	var inflight atomic.Int64
	var maxInflight atomic.Int64

	ForEach(context.Background(), items, func(_ context.Context, _ *int) {
		cur := inflight.Add(1)
		for {
			prev := maxInflight.Load()
			if cur <= prev || maxInflight.CompareAndSwap(prev, cur) {
				break
			}
		}
		inflight.Add(-1)
	})

	if got := maxInflight.Load(); got > DefaultConcurrency {
		t.Errorf("max inflight = %d, want <= %d", got, DefaultConcurrency)
	}
}
