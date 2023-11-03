package integration_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/integration"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
)

func TestCreateVerifyDeleteKeys(t *testing.T) {
	createApiResponse := integration.Step[server.CreateApiResponse]{
		Debug:  true,
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
		require.Equal(t, 200, createApiResponse.Status)
		require.NotEmpty(t, createApiResponse.Header.Get("Unkey-Trace-Id"))
	}()
	// Create 5 keys
	for i := 0; i < 5; i++ {
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
			},
		}.Run(t)
		require.Equal(t, 200, createKeyResponse.Status)
		require.NotEmpty(t, createKeyResponse.Header.Get("Unkey-Trace-Id"))
		require.NotEmpty(t, createKeyResponse.Body.Key)
		require.NotEmpty(t, createKeyResponse.Body.KeyId)

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

		revokeKeyResponse := integration.Step[map[string]any]{
			Name:   "Revoke Key",
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
		require.Equal(t, 200, revokeKeyResponse.Status)
		require.NotEmpty(t, revokeKeyResponse.Header.Get("Unkey-Trace-Id"))

	}

}
