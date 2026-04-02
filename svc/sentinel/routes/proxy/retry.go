package handler

import (
	"errors"
	"io"
	"net"
	"sync/atomic"
	"syscall"
)

// retryBudget prevents retry amplification during outages using a token-bucket.
// Every non-retried request deposits 1 token. Each retry costs retryTokenCost
// tokens, giving an effective retry ratio of 1/retryTokenCost.
type retryBudget struct {
	tokens atomic.Int64
}

const (
	maxRetryTokens     int64 = 500
	retryTokenCost     int64 = 5
	initialRetryTokens int64 = 100
)

func NewRetryBudget() *retryBudget {
	b := &retryBudget{} //nolint:exhaustruct
	b.tokens.Store(initialRetryTokens)
	return b
}

// Deposit adds one token to the bucket (capped at max).
// Called on every non-retried request.
func (b *retryBudget) Deposit() {
	for {
		cur := b.tokens.Load()
		if cur >= maxRetryTokens {
			return
		}
		if b.tokens.CompareAndSwap(cur, cur+1) {
			return
		}
	}
}

// Withdraw attempts to consume retryTokenCost tokens. Returns false if the
// budget is exhausted.
func (b *retryBudget) Withdraw() bool {
	for {
		cur := b.tokens.Load()
		if cur < retryTokenCost {
			return false
		}
		if b.tokens.CompareAndSwap(cur, cur-retryTokenCost) {
			return true
		}
	}
}

// isRetryableError returns true when the error indicates a connection-level
// failure where the backend never processed the request: stale pooled
// connections (EOF), connection resets, connection refused, and host
// unreachable.
func isRetryableError(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return true
		}
		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return true
		}
		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return true
		}
	}

	return false
}
