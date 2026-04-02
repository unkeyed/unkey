package handler

import (
	"errors"
	"io"
	"net"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetryBudget_DepositAndWithdraw(t *testing.T) {
	b := NewRetryBudget()
	// Initial tokens = 100, cost = 5 → 20 withdrawals possible.
	for range 20 {
		require.True(t, b.Withdraw())
	}
	require.False(t, b.Withdraw(), "budget should be exhausted")
}

func TestRetryBudget_DepositReplenishes(t *testing.T) {
	b := NewRetryBudget()
	b.tokens.Store(0)

	// 5 deposits = 5 tokens = 1 withdrawal (cost=5).
	for range 5 {
		b.Deposit()
	}
	require.True(t, b.Withdraw())
	require.False(t, b.Withdraw())
}

func TestRetryBudget_CapsAtMax(t *testing.T) {
	b := NewRetryBudget()
	for range 1000 {
		b.Deposit()
	}
	require.LessOrEqual(t, b.tokens.Load(), maxRetryTokens)
}

func TestRetryBudget_EffectiveRatio(t *testing.T) {
	b := NewRetryBudget()
	b.tokens.Store(0)

	// Simulate mixed traffic: mostly successful, some retries.
	retried := 0
	deposited := 0
	for range 600 {
		if b.Withdraw() {
			retried++
		} else {
			// Non-retried request deposits a token.
			b.Deposit()
			deposited++
		}
	}

	// Effective ratio should be ~1:5 (retried:deposited) = ~16.7%.
	ratio := float64(retried) / float64(deposited)
	require.InDelta(t, 0.2, ratio, 0.05,
		"retry ratio %.2f should be ~20%%", ratio)
}

func TestRetryBudget_ConcurrentSafety(t *testing.T) {
	b := NewRetryBudget()

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				if !b.Withdraw() {
					b.Deposit()
				}
			}
		}()
	}
	wg.Wait()

	tokens := b.tokens.Load()
	require.GreaterOrEqual(t, tokens, int64(0))
	require.LessOrEqual(t, tokens, maxRetryTokens)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"io.EOF", io.EOF, true},
		{"wrapped EOF", errors.Join(errors.New("context"), io.EOF), true},
		{"ECONNRESET", &net.OpError{Op: "read", Err: syscall.ECONNRESET}, true},
		{"ECONNREFUSED", &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}, true},
		{"EHOSTUNREACH", &net.OpError{Op: "dial", Err: syscall.EHOSTUNREACH}, true},
		{"generic error", errors.New("something"), false},
		{"timeout", &net.OpError{Op: "read", Err: &timeoutErr{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.retryable, isRetryableError(tt.err))
		})
	}
}

type timeoutErr struct{}

func (e *timeoutErr) Error() string   { return "timeout" }
func (e *timeoutErr) Timeout() bool   { return true }
func (e *timeoutErr) Temporary() bool { return true }
