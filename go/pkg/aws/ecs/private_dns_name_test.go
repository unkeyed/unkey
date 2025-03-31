package ecs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPrivateDnsName(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/task" {
			w.Write([]byte(`{
                "Containers": [
                    {
                        "Networks": [
                            {
                                "PrivateDNSName": "test-container.local"
                            }
                        ]
                    }
                ]
            }`))
		}
	}))
	defer server.Close()

	// Set the environment variable
	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", server.URL)

	// Call the function
	dnsName, err := GetPrivateDnsName()

	// Check results
	require.NoError(t, err)
	require.Equal(t, "test-container.local", dnsName)
}

// Add a test for the error case
func TestGetPrivateDnsName_Error(t *testing.T) {
	// Clear the environment variable
	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", "")

	// Call the function
	_, err := GetPrivateDnsName()

	// Check results
	require.Error(t, err)
	require.Contains(t, err.Error(), "ECS_CONTAINER_METADATA_URI_V4 is empty")
}
