package vercel

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTransportFailureRetainsSnapshotAndRedacts(t *testing.T) {
	body := []byte(`{"environment":"production","definitions":{"flag":{"variants":[false,true],"environments":{"production":1}}}}`)
	p := testProvider(t, body)
	p.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("vf_server_secret target-id") })
	err := p.refresh(t.Context())
	require.NotContains(t, err.Error(), "secret")
	require.NotContains(t, p.Diagnostics().LastRefreshError, "target-id")
	require.True(t, p.BooleanEvaluation(t.Context(), "flag", false, nil).Value)
}

func TestRefreshRejectsRedirectAndInvalidBodiesWithoutLosingSnapshot(t *testing.T) {
	var redirected atomic.Bool
	destination := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected.Store(true)
	}))
	t.Cleanup(destination.Close)
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusFound)
	}))
	t.Cleanup(source.Close)
	p, err := newProvider(Config{SDKKey: "vf_server_secret", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0}, source.URL, source.Client())
	require.NoError(t, err)
	err = p.refresh(t.Context())
	require.Error(t, err)
	require.False(t, redirected.Load())
	require.Equal(t, "http_302", p.Diagnostics().LastRefreshError)
	require.NotContains(t, err.Error(), "secret")

	valid := []byte(`{"environment":"production","definitions":{"flag":{"variants":[true,false],"environments":{"production":0}}}}`)
	for _, body := range [][]byte{
		[]byte("not json"),
		append(append([]byte{}, valid...), []byte(` {"ignored":true}`)...),
		[]byte(strings.Repeat("x", maxBodyBytes+1)),
	} {
		provider := testProvider(t, valid, body)
		require.Error(t, provider.refresh(t.Context()))
		require.True(t, provider.BooleanEvaluation(t.Context(), "flag", false, nil).Value)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func testProvider(t *testing.T, bodies ...[]byte) *Provider {
	t.Helper()
	index := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := bodies[index]
		index++
		return response(body), nil
	})}
	p, err := newProvider(Config{SDKKey: "vf_server_test", RefreshInterval: time.Second, HTTPTimeout: 0, MaxStaleness: time.Minute}, "https://example.invalid/data", client)
	require.NoError(t, err)
	require.NoError(t, p.refresh(t.Context()))
	return p
}

func response(body []byte) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}
}
