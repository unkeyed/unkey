package certificate

import (
	"context"
	"database/sql"
	"log/slog"
	"reflect"
	"testing"

	"github.com/go-acme/lego/v4/certificate"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// certDB is an in-memory stand-in for the queries ProcessChallenge makes. The
// embedded interface is nil, so an unexpected query panics.
type certDB struct {
	db.Database

	domains []testDomain

	claimedDomainID  string
	verifiedDomainID string
	failedDomainID   string
	inserted         *db.InsertCertificateParams
}

// testDomain is a row in either domain table; the id prefix tells which.
type testDomain struct {
	id          string
	workspaceID string
	domain      string
	verified    bool
}

func (f *certDB) FindVerifiedDomainByHostname(_ context.Context, arg db.FindVerifiedDomainByHostnameParams) (db.FindVerifiedDomainByHostnameRow, error) {
	for _, d := range f.domains {
		if d.domain == arg.Domain && d.verified {
			return db.FindVerifiedDomainByHostnameRow{ID: d.id, WorkspaceID: d.workspaceID, Domain: d.domain}, nil
		}
	}
	return db.FindVerifiedDomainByHostnameRow{}, sql.ErrNoRows //nolint:exhaustruct
}

func (f *certDB) UpdateAcmeChallengeTryClaiming(_ context.Context, arg db.UpdateAcmeChallengeTryClaimingParams) error {
	f.claimedDomainID = arg.DomainID
	return nil
}

func (f *certDB) FindCertificateByHostname(_ context.Context, _ string) (db.Certificate, error) {
	return db.Certificate{}, sql.ErrNoRows //nolint:exhaustruct
}

func (f *certDB) InsertCertificate(_ context.Context, arg db.InsertCertificateParams) error {
	f.inserted = &arg
	return nil
}

func (f *certDB) UpdateAcmeChallengeVerifiedWithExpiry(_ context.Context, arg db.UpdateAcmeChallengeVerifiedWithExpiryParams) error {
	f.verifiedDomainID = arg.DomainID
	return nil
}

func (f *certDB) UpdateAcmeChallengeStatus(_ context.Context, arg db.UpdateAcmeChallengeStatusParams) error {
	f.failedDomainID = arg.DomainID
	return nil
}

type testRunContext struct{ context.Context }

func (testRunContext) Log() *slog.Logger         { return slog.Default() }
func (testRunContext) Request() *restate.Request { return nil }

// newChallengeContext runs every journaled step for real except the ACME
// issuance, which needs a live CA and is replaced by issued.
func newChallengeContext(t *testing.T, issued EncryptedCertificate) *mocks.MockContext {
	t.Helper()

	mockCtx := mocks.NewMockContext(t)
	// The issuance step carries a name plus four retry options.
	mockCtx.EXPECT().
		Run(mock.Anything, mock.AnythingOfType("*certificate.EncryptedCertificate"), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Call.
		Run(func(args mock.Arguments) {
			*args.Get(1).(*EncryptedCertificate) = issued
		}).
		Return(nil).
		Maybe()
	mockCtx.EXPECT().
		Run(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(fn func(restate.RunContext) (any, error), out any, _ ...restate.RunOption) restate.TerminalError {
			value, err := fn(testRunContext{Context: context.Background()})
			if err != nil {
				return restate.ToTerminalError(err)
			}
			reflect.ValueOf(out).Elem().Set(reflect.ValueOf(value))
			return nil
		}).
		Maybe()
	return mockCtx
}

var testIssued = EncryptedCertificate{
	CertificateID:       "cert_test",
	Certificate:         "-----BEGIN CERTIFICATE-----",
	EncryptedPrivateKey: "encrypted",
	ExpiresAt:           1234,
}

// Characterization: a verified deploy domain is claimed, persisted under its own
// workspace, and marked verified by its id.
func TestProcessChallengeDeployDomain(t *testing.T) {
	f := &certDB{ //nolint:exhaustruct
		domains: []testDomain{{id: "dom_deploy", workspaceID: "ws_deploy", domain: "api.example.com", verified: true}},
	}
	resp := processChallenge(t, f, "ws_deploy", "api.example.com")

	require.Equal(t, "success", resp.GetStatus())
	require.Equal(t, "cert_test", resp.GetCertificateId())
	require.Equal(t, "dom_deploy", f.claimedDomainID)
	require.Equal(t, "dom_deploy", f.verifiedDomainID)
	require.Empty(t, f.failedDomainID)
	require.NotNil(t, f.inserted)
	require.Equal(t, "ws_deploy", f.inserted.WorkspaceID)
	require.Equal(t, "api.example.com", f.inserted.Hostname)
	require.Equal(t, "encrypted", f.inserted.EncryptedPrivateKey)
}

// A verified portal domain issues exactly like a deploy domain, under the
// tenant workspace that owns the portal_domains row.
func TestProcessChallengePortalDomain(t *testing.T) {
	f := &certDB{ //nolint:exhaustruct
		domains: []testDomain{{id: "pdom_tenant", workspaceID: "ws_tenant", domain: "portal.example.com", verified: true}},
	}
	resp := processChallenge(t, f, "ws_tenant", "portal.example.com")

	require.Equal(t, "success", resp.GetStatus())
	require.Equal(t, "pdom_tenant", f.claimedDomainID)
	require.Equal(t, "pdom_tenant", f.verifiedDomainID)
	require.NotNil(t, f.inserted)
	require.Equal(t, "ws_tenant", f.inserted.WorkspaceID)
	require.Equal(t, "portal.example.com", f.inserted.Hostname)
}

// A pending competing claim on the hostname is never the row a certificate is
// issued for, even when it is listed first.
func TestProcessChallengeIgnoresPendingClaim(t *testing.T) {
	f := &certDB{ //nolint:exhaustruct
		domains: []testDomain{
			{id: "dom_pending", workspaceID: "ws_pending", domain: "api.example.com", verified: false},
			{id: "pdom_verified", workspaceID: "ws_verified", domain: "api.example.com", verified: true},
		},
	}
	resp := processChallenge(t, f, "ws_pending", "api.example.com")

	require.Equal(t, "success", resp.GetStatus())
	require.Equal(t, "pdom_verified", f.claimedDomainID)
	require.Equal(t, "pdom_verified", f.verifiedDomainID)
	require.Equal(t, "ws_verified", f.inserted.WorkspaceID)
}

// A hostname with no verified row fails terminally before anything is claimed:
// retrying cannot verify it.
func TestProcessChallengeWithoutVerifiedDomainIsTerminal(t *testing.T) {
	f := &certDB{ //nolint:exhaustruct
		domains: []testDomain{{id: "dom_pending", workspaceID: "ws", domain: "api.example.com", verified: false}},
	}
	svc := New(Config{DB: f}) //nolint:exhaustruct

	_, err := svc.ProcessChallenge(restate.WithMockContext(newChallengeContext(t, testIssued)), &hydrav1.ProcessChallengeRequest{
		WorkspaceId: "ws",
		Domain:      "api.example.com",
	})
	require.Error(t, err)
	require.True(t, restate.IsTerminalError(err), "got %v", err)
	require.Empty(t, f.claimedDomainID)
	require.Nil(t, f.inserted)
}

func processChallenge(t *testing.T, f *certDB, workspaceID, domain string) *hydrav1.ProcessChallengeResponse {
	t.Helper()
	svc := New(Config{DB: f}) //nolint:exhaustruct
	resp, err := svc.ProcessChallenge(restate.WithMockContext(newChallengeContext(t, testIssued)), &hydrav1.ProcessChallengeRequest{
		WorkspaceId: workspaceID,
		Domain:      domain,
	})
	require.NoError(t, err)
	return resp
}

// keyringVault records the keyring each private key is encrypted under.
type keyringVault struct {
	vault.VaultServiceClient
	keyrings []string
}

func (v *keyringVault) Encrypt(_ context.Context, req *vaultv1.EncryptRequest) (*vaultv1.EncryptResponse, error) {
	v.keyrings = append(v.keyrings, req.GetKeyring())
	return &vaultv1.EncryptResponse{Encrypted: "encrypted:" + req.GetData(), KeyId: "key"}, nil
}

// The private key is encrypted under the keyring issuance passes, which
// ProcessChallenge sets to the resolved domain row's workspace.
func TestEncryptCertificateUsesKeyring(t *testing.T) {
	v := &keyringVault{VaultServiceClient: nil, keyrings: nil}
	svc := New(Config{Vault: v}) //nolint:exhaustruct

	cert, err := svc.encryptCertificate(context.Background(), "ws_tenant", &certificate.Resource{ //nolint:exhaustruct
		Certificate: []byte("not a pem"),
		PrivateKey:  []byte("private"),
	})
	require.NoError(t, err)
	require.Equal(t, []string{"ws_tenant"}, v.keyrings)
	require.Equal(t, "encrypted:private", cert.EncryptedPrivateKey)
	require.Equal(t, "not a pem", cert.Certificate)
}
