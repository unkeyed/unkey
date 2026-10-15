package certificate

import (
	"context"
	"testing"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestPersistCertificateCannotRecreateDeletedDomainData(t *testing.T) {
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	svc := New(Config{DB: database})

	for _, workspaceID := range []string{uid.New(uid.WorkspacePrefix), "unkey_internal"} {
		t.Run(workspaceID, func(t *testing.T) {
			ctx := t.Context()
			domainID := uid.New(uid.DomainPrefix)
			hostname := uid.DNS1035() + ".example.com"
			_, err := database.RW().ExecContext(ctx, `INSERT INTO custom_domains
				(id, workspace_id, project_id, app_id, environment_id, domain, challenge_type, verification_token, target_cname, created_at)
				VALUES (?, ?, ?, ?, ?, ?, 'HTTP-01', '', ?, 1)`, domainID, workspaceID, workspaceID, workspaceID, domainID, hostname, hostname)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, database.DeleteCustomDomainByID(context.Background(), domainID)) })
			dom, err := database.FindCustomDomainById(ctx, domainID)
			require.NoError(t, err)
			cert := EncryptedCertificate{
				CertificateID: uid.New(uid.CertificatePrefix), Certificate: "first",
				EncryptedPrivateKey: "encrypted", ExpiresAt: 100,
			}
			id, err := svc.persistCertificate(ctx, dom, hostname, cert)
			require.NoError(t, err)
			require.Equal(t, cert.CertificateID, id)
			cert.CertificateID = uid.New(uid.CertificatePrefix)
			cert.Certificate = "renewed"
			renewedID, err := svc.persistCertificate(ctx, dom, hostname, cert)
			require.NoError(t, err)
			require.Equal(t, id, renewedID)
			stored, err := database.FindCertificateByHostname(ctx, hostname)
			require.NoError(t, err)
			require.Equal(t, "renewed", stored.Certificate)

			require.NoError(t, database.DeleteCustomDomainByID(ctx, domainID))
			id, err = svc.persistCertificate(ctx, dom, hostname, cert)
			require.True(t, restate.IsTerminalError(err), "%v", err)
			require.Empty(t, id)
			_, err = database.FindCertificateByHostname(ctx, hostname)
			require.True(t, db.IsNotFound(err), "%v", err)
		})
	}
}
