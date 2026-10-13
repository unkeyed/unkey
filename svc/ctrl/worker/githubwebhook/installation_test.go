package githubwebhook_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	githubhook "github.com/unkeyed/unkey/svc/ctrl/api/webhooks/github"
)

func TestInstallationDeletedDisconnectsAllWorkspaces(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)
	first := h.newTarget(t, ctx)
	second := h.newTarget(t, ctx)
	second.installationID = first.installationID
	other := h.newTarget(t, ctx)
	firstApp := h.newApp(t, ctx, first, appOptions{})
	secondApp := h.newApp(t, ctx, second, appOptions{})
	otherApp := h.newApp(t, ctx, other, appOptions{})
	for _, target := range []deployTarget{first, second, other} {
		require.NoError(t, pkgdb.Query.InsertGithubAppInstallation(ctx, h.database.RW(), pkgdb.InsertGithubAppInstallationParams{
			WorkspaceID: target.workspaceID, InstallationID: target.installationID, CreatedAt: time.Now().UnixMilli(),
		}))
	}
	// A workspace can retain another installation after this one is removed.
	require.NoError(t, pkgdb.Query.InsertGithubAppInstallation(ctx, h.database.RW(), pkgdb.InsertGithubAppInstallationParams{
		WorkspaceID: first.workspaceID, InstallationID: other.installationID, CreatedAt: time.Now().UnixMilli(),
	}))

	secret := uid.New(uid.TestPrefix)
	handler := githubhook.New(h.ingress.IngressClient, secret)
	body, err := json.Marshal(struct {
		Action       string `json:"action"`
		Installation struct {
			ID int64 `json:"id"`
		} `json:"installation"`
	}{Action: "deleted", Installation: struct {
		ID int64 `json:"id"`
	}{ID: first.installationID}})
	require.NoError(t, err)
	mac := hmac.New(sha256.New, []byte(secret))
	_, err = mac.Write(body)
	require.NoError(t, err)
	deliveryID := uid.New(uid.TestPrefix)
	for range 2 {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
		req.Header.Set("X-GitHub-Event", "installation")
		req.Header.Set("X-GitHub-Delivery", deliveryID)
		req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		ids, findErr := pkgdb.Query.FindGithubAppInstallationsByWorkspaceId(ctx, h.database.RO(), first.workspaceID)
		require.NoError(c, findErr)
		require.Equal(c, []int64{other.installationID}, ids)
		ids, findErr = pkgdb.Query.FindGithubAppInstallationsByWorkspaceId(ctx, h.database.RO(), second.workspaceID)
		require.NoError(c, findErr)
		require.Empty(c, ids)
		for _, app := range []targetApp{firstApp, secondApp} {
			_, findErr = h.database.FindGithubRepoConnectionByAppId(ctx, app.id)
			require.ErrorIs(c, findErr, sql.ErrNoRows)
		}
	}, 10*time.Second, 50*time.Millisecond)
	ids, err := pkgdb.Query.FindGithubAppInstallationsByWorkspaceId(ctx, h.database.RO(), other.workspaceID)
	require.NoError(t, err)
	require.Equal(t, []int64{other.installationID}, ids)
	connection, err := h.database.FindGithubRepoConnectionByAppId(ctx, otherApp.id)
	require.NoError(t, err)
	require.Equal(t, other.installationID, connection.InstallationID)
}
