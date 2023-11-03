package integration_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/integration"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
)

func TestExceedUsage(t *testing.T) {
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
			Url:    fmt.Sprintf("%s/v1/apis.deleteApi", BASE_URL),
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
			"apiId":     createApiResponse.Body.ApiId,
			"remaining": 5,
		},
	}.Run(t)
	require.Equal(t, 200, createKeyResponse.Status)
	require.NotEmpty(t, createKeyResponse.Body.Key)
	require.NotEmpty(t, createKeyResponse.Body.KeyId)
	require.NotEmpty(t, createKeyResponse.Header.Get("Unkey-Trace-Id"))

	// Use up all 5 verifications
	for i := 4; i >= 0; i-- {

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
		require.Equal(t, int32(i), *verifyKeyResponse.Body.Remaining)
		require.NotEmpty(t, verifyKeyResponse.Header.Get("Unkey-Trace-Id"))
	}
	failingVerifyKeyResponse := integration.Step[server.VerifyKeyResponseV1]{
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
	require.Equal(t, 200, failingVerifyKeyResponse.Status)
	require.False(t, failingVerifyKeyResponse.Body.Valid)
	require.Equal(t, int32(0), *failingVerifyKeyResponse.Body.Remaining)
	require.Equal(t, "KEY_USAGE_EXCEEDED", failingVerifyKeyResponse.Body.Code)
	require.NotEmpty(t, failingVerifyKeyResponse.Header.Get("Unkey-Trace-Id"))

}
