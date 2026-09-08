package validation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaIsValid(t *testing.T) {
	_, err := New()
	require.NoError(t, err)
}

func TestConcurrentCreateKeyValidation(t *testing.T) {
	v, err := New()
	require.NoError(t, err)
	for range 100 {
		t.Run("request", func(t *testing.T) {
			t.Parallel()
			for range 100 {
				r := httptest.NewRequest(http.MethodPost, "/v2/keys.createKey", strings.NewReader(`{"apiId":"api_test","externalId":"user_concurrent_test"}`))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer test")
				response, valid := v.Validate(context.Background(), r)
				require.True(t, valid, "%+v", response)
			}
		})
	}
}

func BenchmarkCreateKeyValidation(b *testing.B) {
	v, err := New()
	require.NoError(b, err)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := httptest.NewRequest(http.MethodPost, "/v2/keys.createKey", strings.NewReader(`{"apiId":"api_test","externalId":"user_concurrent_test"}`))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Authorization", "Bearer test")
			response, valid := v.Validate(context.Background(), r)
			if !valid {
				b.Errorf("unexpected validation failure: %+v", response)
			}
		}
	})
}
