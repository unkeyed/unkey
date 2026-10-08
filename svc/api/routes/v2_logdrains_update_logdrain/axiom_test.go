package logdrains_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"google.golang.org/protobuf/proto"
)

func TestUpdateAxiomPartialDestinationPreservesOmittedFields(t *testing.T) {
	for _, field := range []string{"dataset", "token"} {
		t.Run(field+" only", func(t *testing.T) {
			h, route, id, expected := seedUpdateDrain(t)
			workspaceID := h.Resources().UserWorkspace.ID
			encrypted, err := h.Vault.Encrypt(t.Context(), &vaultv1.EncryptRequest{
				Keyring: workspaceID,
				Data:    "original-token",
			})
			require.NoError(t, err)
			expected.Destination = &logdrainv1.Config_Axiom{Axiom: &logdrainv1.AxiomConfig{
				Dataset:        "original-dataset",
				EncryptedToken: encrypted.GetEncrypted(),
			}}
			encoded, err := proto.Marshal(expected)
			require.NoError(t, err)
			_, err = h.DB.RW().ExecContext(t.Context(), "UPDATE logdrains SET config = ? WHERE id = ?", encoded, id)
			require.NoError(t, err)
			key := h.CreateRootKey(workspaceID, urn.New().Workspace(workspaceID).Logdrain(id).String()+"#write")
			res := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, route, updateHeaders(key), json.RawMessage(`{"logdrainId":"`+id+`","destination":{"axiom":{"`+field+`":"replacement"}}}`))
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			require.Equal(t, id, res.Body.Data.Id)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.NotContains(t, res.RawBody, "token")
			plaintext := "original-token"
			if field == "dataset" {
				expected.GetAxiom().Dataset = "replacement"
			} else {
				var stored logdrainv1.Config
				require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT config FROM logdrains WHERE id = ?", id).Scan(&encoded))
				require.NoError(t, proto.Unmarshal(encoded, &stored))
				require.NotEqual(t, encrypted.GetEncrypted(), stored.GetAxiom().EncryptedToken)
				expected.GetAxiom().EncryptedToken = stored.GetAxiom().EncryptedToken
				plaintext = "replacement"
			}
			secret, err := h.Vault.Decrypt(t.Context(), &vaultv1.DecryptRequest{
				Keyring:   workspaceID,
				Encrypted: expected.GetAxiom().EncryptedToken,
			})
			require.NoError(t, err)
			require.Equal(t, plaintext, secret.GetPlaintext())
			assertUpdateState(t, h, id, "Original", expected, true)
		})
	}
}
