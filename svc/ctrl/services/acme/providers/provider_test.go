package providers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type canceledLookup struct {
	db.Database
}

func (c canceledLookup) FindCustomDomainByDomainOrWildcard(ctx context.Context, _ db.FindCustomDomainByDomainOrWildcardParams) (db.CustomDomain, error) {
	return db.CustomDomain{}, ctx.Err()
}

func TestPresentPropagatesCancellation(t *testing.T) {
	c, err := cache.New(cache.Config[string, db.CustomDomain]{
		Fresh:    time.Minute,
		Stale:    2 * time.Minute,
		MaxSize:  10,
		Resource: t.Name(),
		Clock:    clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	p := Provider{db: canceledLookup{Database: nil}, dns: nil, cache: c}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, p.Present(ctx, "example.com", "token", "authorization"), context.Canceled)
}

type cleanupDNS struct {
	DNSProvider
	cleanup func(context.Context, string, string, string) error
}

func (d cleanupDNS) CleanUp(ctx context.Context, domain, token, keyAuth string) error {
	return d.cleanup(ctx, domain, token, keyAuth)
}

func TestCleanupSurvivesCanceledIssuance(t *testing.T) {
	called := false
	p := Provider{
		db:    nil,
		cache: nil,
		dns: cleanupDNS{
			DNSProvider: nil,
			cleanup: func(ctx context.Context, domain, token, keyAuth string) error {
				called = true
				require.NoError(t, ctx.Err())
				require.Equal(t, "example.com", domain)
				require.Equal(t, "token", token)
				require.Equal(t, "authorization", keyAuth)
				return nil
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, p.CleanUp(ctx, "example.com", "token", "authorization"))
	require.True(t, called)
}
