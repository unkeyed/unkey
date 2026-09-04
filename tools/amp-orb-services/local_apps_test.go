package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRewriteHostHeaderReplacesExistingHost(t *testing.T) {
	t.Parallel()

	request := []byte("GET /health HTTP/1.1\r\nHost: 127.0.0.1:20000\r\nAccept: */*\r\n\r\n")
	got := rewriteHostHeader(request, "demo.unkey.local")

	require.Equal(t, "GET /health HTTP/1.1\r\nHost: demo.unkey.local\r\nAccept: */*\r\n\r\n", string(got))
}

func TestRewriteHostHeaderInsertsMissingHost(t *testing.T) {
	t.Parallel()

	request := []byte("GET / HTTP/1.1\r\nAccept: */*\r\n\r\n")
	got := rewriteHostHeader(request, "demo.unkey.local")

	require.Equal(t, "GET / HTTP/1.1\r\nHost: demo.unkey.local\r\nAccept: */*\r\n\r\n", string(got))
}

func TestAllocateLocalAppPortUsesDeclaredRange(t *testing.T) {
	t.Parallel()

	used := map[int]struct{}{localAppPortStart: {}}
	port, err := allocateLocalAppPort("d_test", used)
	require.NoError(t, err)
	require.GreaterOrEqual(t, port, localAppPortStart)
	require.Less(t, port, localAppPortStart+localAppPortCount)
}

func TestAllocateLocalAppPortStableForDeployment(t *testing.T) {
	t.Parallel()

	first, err := allocateLocalAppPort("d_stable", map[int]struct{}{})
	require.NoError(t, err)
	second, err := allocateLocalAppPort("d_stable", map[int]struct{}{})
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestAppPublicURLRewritesHostPort(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "http://localhost:19081/", nil)
	require.Equal(t, "http://localhost:20000/", appPublicURL(request, 20000))
}

func TestAppPublicURLKeepsLoopbackWhenHostHasNoPort(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "https://apps.example.com/", nil)
	require.Equal(t, "http://127.0.0.1:20000/", appPublicURL(request, 20000))
}
