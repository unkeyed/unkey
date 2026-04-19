package proxy

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
)

func newTestService(t *testing.T) *service {
	t.Helper()
	svc, err := New(Config{
		InstanceID: "test-instance",
		Platform:   "dev",
		Region:     "local",
		ApexDomain: "unkey.cloud",
		Clock:      clock.New(),
		MaxHops:    3,
	})
	require.NoError(t, err)
	return svc
}

func newTestSession(t *testing.T, method, path, host string) *zen.Session {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Host = host
	w := httptest.NewRecorder()
	sess := &zen.Session{}
	err := sess.Init(w, req, 0)
	require.NoError(t, err)
	return sess
}
