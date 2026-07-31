package singleflight_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/singleflight"
)

func TestGroup(t *testing.T) {
	t.Run("coalesces concurrent calls with the same structured key", func(t *testing.T) {
		t.Parallel()

		type key struct {
			workspaceID string
			keyID       string
		}

		var group singleflight.Group[key, string]
		var calls atomic.Int64
		started := make(chan struct{})
		release := make(chan struct{})

		fn := func() (string, error) {
			if calls.Add(1) == 1 {
				close(started)
			}
			<-release
			return "value", nil
		}

		const goroutines = 10
		results := make(chan string, goroutines)
		errs := make(chan error, goroutines)
		var ready sync.WaitGroup
		ready.Add(goroutines)
		start := make(chan struct{})
		calling := make(chan struct{}, goroutines)

		for range goroutines {
			go func() {
				ready.Done()
				<-start
				calling <- struct{}{}
				value, err := group.Do(key{workspaceID: "ws_123", keyID: "key_123"}, fn)
				results <- value
				errs <- err
			}()
		}

		ready.Wait()
		close(start)
		<-started
		for range goroutines {
			<-calling
		}
		// Give every caller an opportunity to enter Do before releasing the
		// shared function.
		time.Sleep(10 * time.Millisecond)
		close(release)

		for range goroutines {
			require.NoError(t, <-errs)
			require.Equal(t, "value", <-results)
		}
		require.Equal(t, int64(1), calls.Load())
	})

	t.Run("does not coalesce different structured keys", func(t *testing.T) {
		t.Parallel()

		type key struct {
			left  string
			right string
		}

		var group singleflight.Group[key, string]
		firstStarted := make(chan struct{})
		releaseFirst := make(chan struct{})
		type result struct {
			value string
			err   error
		}
		firstResult := make(chan result, 1)

		go func() {
			value, err := group.Do(key{left: "a:b", right: "c"}, func() (string, error) {
				close(firstStarted)
				<-releaseFirst
				return "first", nil
			})
			firstResult <- result{value: value, err: err}
		}()

		<-firstStarted
		secondResult := make(chan result, 1)
		go func() {
			value, err := group.Do(key{left: "a", right: "b:c"}, func() (string, error) {
				return "second", nil
			})
			secondResult <- result{value: value, err: err}
		}()

		select {
		case second := <-secondResult:
			require.NoError(t, second.err)
			require.Equal(t, "second", second.value)
		case <-time.After(time.Second):
			close(releaseFirst)
			t.Fatal("different keys were coalesced")
		}

		close(releaseFirst)
		first := <-firstResult
		require.NoError(t, first.err)
		require.Equal(t, "first", first.value)
	})

	t.Run("runs again after a flight completes", func(t *testing.T) {
		t.Parallel()

		var group singleflight.Group[string, int]
		var calls int

		first, err := group.Do("key", func() (int, error) {
			calls++
			return calls, nil
		})
		require.NoError(t, err)

		second, err := group.Do("key", func() (int, error) {
			calls++
			return calls, nil
		})
		require.NoError(t, err)

		require.Equal(t, 1, first)
		require.Equal(t, 2, second)
	})
}
