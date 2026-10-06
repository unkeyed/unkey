package domainverify

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const target = "abc123.cname.unkey.local"

type fakeResolver struct {
	cname string
	txt   []string
	hosts []string
	err   error
}

func notFound(name string) error {
	return &net.DNSError{Err: "no such host", Name: name, Server: "", IsTimeout: false, IsTemporary: false, IsNotFound: true, UnwrapErr: nil}
}

func (r fakeResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	if len(r.txt) == 0 {
		return nil, notFound(name)
	}
	return r.txt, nil
}

func (r fakeResolver) LookupCNAME(_ context.Context, name string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	if r.cname == "" {
		return "", notFound(name)
	}
	return r.cname, nil
}

func (r fakeResolver) LookupHost(_ context.Context, name string) ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	if len(r.hosts) == 0 {
		return nil, notFound(name)
	}
	return r.hosts, nil
}

// fakeClaims answers the contention lookup and counts calls, so tests can pin
// when it is skipped.
type fakeClaims struct {
	contested bool
	err       error
	calls     int
}

func (f *fakeClaims) FindVerifiedDomainClaimExcludingWorkspace(_ context.Context, arg db.FindVerifiedDomainClaimExcludingWorkspaceParams) (db.FindVerifiedDomainClaimExcludingWorkspaceRow, error) {
	f.calls++
	if f.err != nil {
		return db.FindVerifiedDomainClaimExcludingWorkspaceRow{}, f.err //nolint:exhaustruct
	}
	if !f.contested {
		return db.FindVerifiedDomainClaimExcludingWorkspaceRow{}, sql.ErrNoRows //nolint:exhaustruct
	}
	return db.FindVerifiedDomainClaimExcludingWorkspaceRow{Source: string(ClaimSourceCustom), ID: "dom_other", WorkspaceID: "ws_other"}, nil
}

func domain(hostname string) Domain {
	return Domain{Hostname: hostname, WorkspaceID: "ws_self", TargetCname: target, VerificationToken: "tok"}
}

const proof = "unkey-domain-verify=tok"

func TestDecide(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   Outcome
		want bool
	}{
		{"cname alone", Outcome{CnameVerified: true, TxtVerified: true}, true},                      //nolint:exhaustruct
		{"txt alone", Outcome{CnameVerified: false, TxtVerified: true}, true},                       //nolint:exhaustruct
		{"neither", Outcome{CnameVerified: false, TxtVerified: false}, false},                       //nolint:exhaustruct
		{"contested without txt", Outcome{CnameVerified: true, Contested: true}, false},             //nolint:exhaustruct
		{"contested with txt", Outcome{TxtVerified: true, Contested: true}, true},                   //nolint:exhaustruct
		{"apex without address", Outcome{TxtVerified: true, IsApex: true}, false},                   //nolint:exhaustruct
		{"apex with address", Outcome{TxtVerified: true, IsApex: true, ApexHasRecords: true}, true}, //nolint:exhaustruct
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, decide(tc.in))
		})
	}
}

func TestCheckCNAMEWithoutContentionNeedsNoTXT(t *testing.T) {
	claims := &fakeClaims{contested: false, err: nil, calls: 0}
	out, err := Check(context.Background(), fakeResolver{cname: target, txt: nil, hosts: nil, err: nil}, claims, domain("api.example.com"))
	require.NoError(t, err)
	require.True(t, out.Verified)
	require.False(t, out.RequiresTxt)
	require.True(t, out.TxtVerified, "ownership_verified has always read true when TXT was not required")
}

func TestCheckContentionRequiresTXT(t *testing.T) {
	claims := &fakeClaims{contested: true, err: nil, calls: 0}

	out, err := Check(context.Background(), fakeResolver{cname: target, txt: nil, hosts: nil, err: nil}, claims, domain("api.example.com"))
	require.NoError(t, err)
	require.True(t, out.Contested)
	require.False(t, out.Verified)

	out, err = Check(context.Background(), fakeResolver{cname: target, txt: []string{proof}, hosts: nil, err: nil}, claims, domain("api.example.com"))
	require.NoError(t, err)
	require.True(t, out.Verified)
}

// Without a matching CNAME TXT is already required, so the lookup would add a
// query per retry and change nothing.
func TestCheckSkipsContentionLookupWithoutCNAME(t *testing.T) {
	claims := &fakeClaims{contested: true, err: nil, calls: 0}
	out, err := Check(context.Background(), fakeResolver{cname: "", txt: []string{proof}, hosts: nil, err: nil}, claims, domain("api.example.com"))
	require.NoError(t, err)
	require.True(t, out.Verified)
	require.Zero(t, claims.calls)
}

func TestCheckCNAMEToAnotherTargetDoesNotVerify(t *testing.T) {
	claims := &fakeClaims{contested: false, err: nil, calls: 0}
	out, err := Check(context.Background(), fakeResolver{cname: "someone-else.cname.unkey.local", txt: nil, hosts: nil, err: nil}, claims, domain("api.example.com"))
	require.NoError(t, err)
	require.False(t, out.CnameVerified)
	require.False(t, out.Verified)
}

func TestCheckApexNeedsAddressRecords(t *testing.T) {
	claims := &fakeClaims{contested: false, err: nil, calls: 0}

	out, err := Check(context.Background(), fakeResolver{cname: "", txt: []string{proof}, hosts: nil, err: nil}, claims, domain("example.com"))
	require.NoError(t, err)
	require.True(t, out.IsApex)
	require.False(t, out.Verified)

	out, err = Check(context.Background(), fakeResolver{cname: "", txt: []string{proof}, hosts: []string{"192.0.2.1"}, err: nil}, claims, domain("example.com"))
	require.NoError(t, err)
	require.True(t, out.Verified)
}

// A resolver failure reads as a missing record so the workflow retries rather
// than failing; only a database failure is returned.
func TestCheckDNSErrorIsNotVerifiedButDatabaseErrorFails(t *testing.T) {
	out, err := Check(context.Background(), fakeResolver{cname: "", txt: nil, hosts: nil, err: errors.New("timeout")}, &fakeClaims{contested: false, err: nil, calls: 0}, domain("api.example.com"))
	require.NoError(t, err)
	require.False(t, out.Verified)

	_, err = Check(context.Background(), fakeResolver{cname: target, txt: nil, hosts: nil, err: nil}, &fakeClaims{contested: false, err: errors.New("db down"), calls: 0}, domain("api.example.com"))
	require.Error(t, err)
}
