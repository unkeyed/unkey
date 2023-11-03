package integration_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/integration"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
)

func TestListKeys(t *testing.T) {
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

		deleteApiresponse := integration.Step[map[string]any]{
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
		require.Equal(t, 200, deleteApiresponse.Status)
		require.NotEmpty(t, deleteApiresponse.Header.Get("Unkey-Trace-Id"))
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
		require.NotEmpty(t, createKeyResponse.Body.Key)
		require.NotEmpty(t, createKeyResponse.Body.KeyId)
		require.NotEmpty(t, createKeyResponse.Header.Get("Unkey-Trace-Id"))

		defer func() {

			revokeKeyResponse := integration.Step[map[string]any]{
				Name:   "Revoke Key",
				Method: "POST",
				Url:    fmt.Sprintf("%s/v1/keys.deleteKey", BASE_URL),
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
		}()

	}

	listKeysResponse := integration.Step[server.ListKeysResponse]{
		Name:   "List Keys",
		Method: "GET",
		Url:    fmt.Sprintf("%s/v1/apis/%s/keys", BASE_URL, createApiResponse.Body.ApiId),
		Header: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", ROOT_KEY),
		},
	}.Run(t)
	require.Equal(t, 200, listKeysResponse.Status)
	require.NotEmpty(t, listKeysResponse.Header.Get("Unkey-Trace-Id"))
	require.Equal(t, 5, len(listKeysResponse.Body.Keys))

}
