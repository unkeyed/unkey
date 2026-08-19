package acme

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

// TestVerifyCertificateRejectsMissingBearerToken guarantees unauthenticated
// callers cannot use the ACME verification endpoint to enumerate domains or
// populate ctrl caches.
func TestVerifyCertificateRejectsMissingBearerToken(t *testing.T) {
	svc := New(Config{
		DB:             nil,
		DomainCache:    nil,
		ChallengeCache: nil,
		Bearer:         "ctrl-token",
	})
	req := connect.NewRequest(&ctrlv1.VerifyCertificateRequest{
		Domain: "api.example.com",
		Token:  "challenge-token",
	})

	_, err := svc.VerifyCertificate(context.Background(), req)

	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// TestVerifyCertificateRejectsInvalidBearerToken guarantees network access to
// ctrl is not enough to invoke ACME verification without the preshared token.
func TestVerifyCertificateRejectsInvalidBearerToken(t *testing.T) {
	svc := New(Config{
		DB:             nil,
		DomainCache:    nil,
		ChallengeCache: nil,
		Bearer:         "ctrl-token",
	})
	req := connect.NewRequest(&ctrlv1.VerifyCertificateRequest{
		Domain: "api.example.com",
		Token:  "challenge-token",
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	_, err := svc.VerifyCertificate(context.Background(), req)

	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
