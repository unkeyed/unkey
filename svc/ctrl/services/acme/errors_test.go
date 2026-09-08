package acme

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	legoacme "github.com/go-acme/lego/v5/acme"
	"github.com/stretchr/testify/require"
)

func TestRateLimitRetryAfter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var problem legoacme.ProblemDetails
		problem.Type = "urn:ietf:params:acme:error:rateLimited"
		problem.Detail = "too many certificates"
		for _, delay := range []time.Duration{0, 5 * time.Minute} {
			var rateLimitError error = &legoacme.RateLimitedError{
				ProblemDetails: &problem,
				RetryAfter:     delay,
			}
			err := fmt.Errorf("obtain certificate: %w", rateLimitError)
			parsed := ParseACMEError(err)
			require.Equal(t, ACMEErrorRateLimited, parsed.Type)
			require.False(t, parsed.IsRetryable)
			if delay == 0 {
				require.True(t, parsed.RetryAfter.IsZero())
			} else {
				require.Equal(t, time.Now().Add(delay), parsed.RetryAfter)
			}
		}
	})
}
