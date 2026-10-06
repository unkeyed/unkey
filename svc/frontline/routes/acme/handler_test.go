package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

// stubAcme answers every token with a fixed authorization.
type stubAcme struct {
	ctrl.AcmeServiceClient
}

func (stubAcme) VerifyCertificate(_ context.Context, req *ctrlv1.VerifyCertificateRequest) (*ctrlv1.VerifyCertificateResponse, error) {
	return &ctrlv1.VerifyCertificateResponse{Authorization: "auth-" + req.GetToken()}, nil
}

// The handler answers HTTP-01 for hostnames registered in either domain table,
// so it runs against real MySQL to cover the lookup query.
func TestHandleChecksBothDomainTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := containers.MySQLIsolated(t).DSN
	querier, closeDB, err := db.New(dsn, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeDB()) })

	seed, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, seed.Close()) })

	now := time.Now().UnixMilli()
	_, err = seed.Exec(
		"INSERT INTO portal_domains (id, workspace_id, portal_id, domain, verification_token, target_cname, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		uid.New(uid.PortalDomainPrefix), uid.New(uid.WorkspacePrefix), uid.New(uid.PortalPrefix), "portal.example.com", uid.New(uid.TestPrefix), uid.New(uid.TestPrefix), now,
	)
	require.NoError(t, err)
	_, err = seed.Exec(
		"INSERT INTO custom_domains (id, workspace_id, project_id, app_id, environment_id, domain, challenge_type, verification_token, target_cname, created_at) VALUES (?, ?, ?, ?, ?, ?, 'HTTP-01', ?, ?, ?)",
		uid.New(uid.DomainPrefix), uid.New(uid.WorkspacePrefix), uid.New(uid.ProjectPrefix), uid.New(uid.AppPrefix), uid.New(uid.EnvironmentPrefix), "deploy.example.com", uid.New(uid.TestPrefix), uid.New(uid.TestPrefix), now,
	)
	require.NoError(t, err)

	h := &Handler{AcmeClient: stubAcme{AcmeServiceClient: nil}, DB: querier}
	handle := func(host string) (*httptest.ResponseRecorder, error) {
		req := httptest.NewRequest(http.MethodGet, "http://"+host+"/.well-known/acme-challenge/tok", nil)
		rec := httptest.NewRecorder()
		sess := new(zen.Session)
		require.NoError(t, sess.Init(rec, req, 0))
		return rec, h.Handle(context.Background(), sess)
	}

	for _, host := range []string{"portal.example.com", "deploy.example.com"} {
		t.Run(host, func(t *testing.T) {
			rec, err := handle(host)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, "auth-tok", rec.Body.String())
		})
	}

	t.Run("a hostname in neither table is not configured", func(t *testing.T) {
		_, err := handle("unknown.example.com")
		require.Error(t, err)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.Frontline.Routing.ConfigNotFound.URN(), code)
	})
}
