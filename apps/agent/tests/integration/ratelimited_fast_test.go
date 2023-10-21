package integration_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/integration"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
)

func TestRatelimited(t *testing.T) {

	createApiResponse := integration.Step[server.CreateApiResponse]{
		Name:   "Create API",
		Method: "POST",
		Url:    fmt.Sprintf("%s/v1/apis.createApi", BASE_URL),
		Header: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", ROOT_KEY),
		},
		Body: map[string]any{
			"name": "scenario-test-pls-delete",
		},
	}.Run(t)
	require.Equal(t, 200, createApiResponse.Status)
	require.NotEmpty(t, createApiResponse.Body.ApiId)
	require.NotEmpty(t, createApiResponse.Header.Get("Unkey-Trace-Id"))

	defer func() {
		deleteApiResponse := integration.Step[map[string]any]{
			Name:   "Delete API",
			Method: "POST",
			Url:    fmt.Sprintf("%s/v1/apis.removeApi", BASE_URL),
			Header: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": fmt.Sprintf("Bearer %s", ROOT_KEY),
			},
			Body: map[string]any{
				"apiId": createApiResponse.Body.ApiId,
			},
		}.Run(t)
		require.Equal(t, 200, deleteApiResponse.Status)
		require.NotEmpty(t, deleteApiResponse.Header.Get("Unkey-Trace-Id"))
	}()

	createKeyResponse := integration.Step[server.CreateKeyResponse]{
		Name:   "Create Key",
		Method: "POST",
		Url:    fmt.Sprintf("%s/v1/keys.createKey", BASE_URL),
		Header: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", ROOT_KEY),
		},
		Body: map[string]any{
			"apiId": createApiResponse.Body.ApiId,
			"ratelimit": map[string]any{
				"type":           "fast",
				"limit":          5,
				"refillRate":     1,
				"refillInterval": 5000,
			},
		},
	}.Run(t)
	require.Equal(t, 200, createKeyResponse.Status)
	require.NotEmpty(t, createKeyResponse.Body.Key)
	require.NotEmpty(t, createKeyResponse.Body.KeyId)
	require.NotEmpty(t, createKeyResponse.Header.Get("Unkey-Trace-Id"))

	defer func() {
		removeKeyResponse := integration.Step[map[string]any]{
			Name:   "Remove Key",
			Method: "POST",
			Url:    fmt.Sprintf("%s/v1/keys.removeKey", BASE_URL),
			Header: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": fmt.Sprintf("Bearer %s", ROOT_KEY),
			},
			Body: map[string]any{
				"keyId": createKeyResponse.Body.KeyId,
			},
		}.Run(t)
		require.Equal(t, 200, removeKeyResponse.Status)
		require.NotEmpty(t, removeKeyResponse.Header.Get("Unkey-Trace-Id"))
	}()

	n := 100
	success := 0
	blocked := 0

	wg := sync.WaitGroup{}
	wg.Add(n)
	// Use up all 5 verifications
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			verifyResponse := integration.Step[server.VerifyKeyResponseV1]{
				Name:   "Verify Key",
				Method: "POST",
				Url:    fmt.Sprintf("%s/v1/keys.verifyKey", BASE_URL),
				Header: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]any{
					"key": createKeyResponse.Body.Key,
				},
			}.Run(t)
			require.Equal(t, 200, verifyResponse.Status)
			require.NotEmpty(t, verifyResponse.Header.Get("Unkey-Trace-Id"))
			if verifyResponse.Body.Valid {
				success++
			} else {
				blocked++
			}
		}()
	}
	wg.Wait()

	require.LessOrEqual(t, success, 10)
	require.Equal(t, 100-success, blocked)

	failingVerifykeyResponse := integration.Step[server.VerifyKeyResponseV1]{
		Name:   "Verify Key - should fail",
		Method: "POST",
		Url:    fmt.Sprintf("%s/v1/keys.verifyKey", BASE_URL),
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: map[string]any{
			"key": createKeyResponse.Body.Key,
		},
	}.Run(t)
	require.Equal(t, 200, failingVerifykeyResponse.Status)
	require.False(t, failingVerifykeyResponse.Body.Valid)
	require.Equal(t, "RATELIMITED", failingVerifykeyResponse.Body.Code)
	require.NotEmpty(t, failingVerifykeyResponse.Header.Get("Unkey-Trace-Id"))

	time.Sleep(time.Second * 5)

	verifyKeyResponse := integration.Step[server.VerifyKeyResponseV1]{
		Name:   "Verify Key",
		Method: "POST",
		Url:    fmt.Sprintf("%s/v1/keys.verifyKey", BASE_URL),
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: map[string]any{
			"key": createKeyResponse.Body.Key,
		},
	}.Run(t)
	require.Equal(t, 200, verifyKeyResponse.Status)
	require.True(t, verifyKeyResponse.Body.Valid)
	require.NotEmpty(t, verifyKeyResponse.Header.Get("Unkey-Trace-Id"))

}
