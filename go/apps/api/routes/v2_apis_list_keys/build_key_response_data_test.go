package handler_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_list_keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestJSONUnmarshalingErrors(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
		Vault:  h.Vault,
	}

	t.Run("Invalid identity meta JSON", func(t *testing.T) {
		// Create KeyData with invalid JSON in identity meta
		keyData := &db.KeyData{
			Key: db.Key{
				ID:         "test_key_123",
				CreatedAtM: time.Now().UnixMilli(),
				Enabled:    true,
				Start:      "test_",
			},
			Identity: &db.Identity{
				ID:         "identity_123",
				ExternalID: "external_123",
				Meta:       []byte(`{"invalid": json}`), // Invalid JSON
			},
		}

		_, err := route.BuildKeyResponseData(keyData, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to unmarshal")
	})

	t.Run("Invalid key meta JSON", func(t *testing.T) {
		keyData := &db.KeyData{
			Key: db.Key{
				ID:         "test_key_456",
				CreatedAtM: time.Now().UnixMilli(),
				Enabled:    true,
				Start:      "test_",
				Meta:       sql.NullString{Valid: true, String: `{"broken": syntax}`}, // Invalid JSON
			},
		}

		_, err := route.BuildKeyResponseData(keyData, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to unmarshal")
	})

	t.Run("Valid JSON - should succeed", func(t *testing.T) {
		keyData := &db.KeyData{
			Key: db.Key{
				ID:         "test_key_789",
				CreatedAtM: time.Now().UnixMilli(),
				Enabled:    true,
				Start:      "test_",
				Meta:       sql.NullString{Valid: true, String: `{"environment": "production"}`},
			},
			Identity: &db.Identity{
				ID:         "identity_789",
				ExternalID: "external_789",
				Meta:       []byte(`{"country": "US"}`),
			},
		}

		response, err := route.BuildKeyResponseData(keyData, "")
		require.NoError(t, err)
		require.Equal(t, "test_key_789", response.KeyId)
		require.NotNil(t, response.Meta)
		require.NotNil(t, response.Identity)
		require.NotNil(t, response.Identity.Meta)
	})

	t.Run("Empty meta should not error", func(t *testing.T) {
		keyData := &db.KeyData{
			Key: db.Key{
				ID:         "test_key_empty",
				CreatedAtM: time.Now().UnixMilli(),
				Enabled:    true,
				Start:      "test_",
				Meta:       sql.NullString{Valid: false}, // No meta
			},
			Identity: &db.Identity{
				ID:         "identity_empty",
				ExternalID: "external_empty",
				Meta:       []byte{}, // Empty meta
			},
		}

		response, err := route.BuildKeyResponseData(keyData, "")
		require.NoError(t, err)
		require.Equal(t, "test_key_empty", response.KeyId)
		require.Nil(t, response.Meta)
		require.NotNil(t, response.Identity)
		require.Nil(t, response.Identity.Meta) // Should be nil for empty meta
	})
}
