package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestInstallationWebhookRequiresDurableAcceptanceOnlyForDeletion(t *testing.T) {
	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(unavailable.Close)
	secret := uid.New(uid.TestPrefix)
	handler := New(ingress.NewClient(unavailable.URL), secret)
	for _, tt := range []struct {
		action string
		status int
	}{
		{action: "deleted", status: http.StatusInternalServerError},
		{action: "created", status: http.StatusOK},
		{action: "suspend", status: http.StatusOK},
		{action: "unsuspend", status: http.StatusOK},
		{action: "new_permissions_accepted", status: http.StatusOK},
	} {
		t.Run(tt.action, func(t *testing.T) {
			body, err := json.Marshal(installationPayload{
				Action: tt.action, Installation: pushInstallation{ID: time.Now().UnixNano()},
			})
			require.NoError(t, err)
			mac := hmac.New(sha256.New, []byte(secret))
			_, err = mac.Write(body)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
			req.Header.Set("X-GitHub-Event", "installation")
			req.Header.Set("X-GitHub-Delivery", uid.New(uid.TestPrefix))
			req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			require.Equal(t, tt.status, res.Code, res.Body.String())
		})
	}
}
