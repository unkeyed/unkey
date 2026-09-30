package vercel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/featureflag"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestReferenceFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/direct-targets.json")
	require.NoError(t, err)
	s, err := parseDatafile(body, time.Now())
	require.NoError(t, err)
	p := &Provider{config: Config{MaxStaleness: time.Minute}}
	p.snapshot.Store(s)

	var tests []struct {
		Flag     string                       `json:"flag"`
		Entities openfeature.FlattenedContext `json:"entities"`
		Value    bool                         `json:"value"`
		Reason   string                       `json:"reason"`
	}
	expected, err := os.ReadFile("testdata/expected.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(expected, &tests))
	require.NotEmpty(t, tests)
	reasons := map[string]openfeature.Reason{
		"target_match": openfeature.TargetingMatchReason,
		"fallthrough":  openfeature.DefaultReason,
		"paused":       openfeature.StaticReason,
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%s/%d", test.Flag, i), func(t *testing.T) {
			detail := p.BooleanEvaluation(context.Background(), test.Flag, !test.Value, test.Entities)
			require.Equal(t, test.Value, detail.Value)
			require.NoError(t, detail.Error())
			require.Contains(t, reasons, test.Reason)
			require.Equal(t, reasons[test.Reason], detail.Reason)
		})
	}
}

func TestErrorsReturnAsymmetricDefaults(t *testing.T) {
	p := &Provider{config: Config{MaxStaleness: time.Minute}}
	p.snapshot.Store(&snapshot{fetchedAt: time.Now(), flags: map[string]compiledFlag{
		"unsupported": {err: errors.New("rules are unsupported")},
	}})

	for _, defaultValue := range []bool{false, true} {
		missing := p.BooleanEvaluation(context.Background(), "missing", defaultValue, nil)
		require.Equal(t, defaultValue, missing.Value)
		require.Equal(t, openfeature.FlagNotFoundCode, missing.ResolutionDetail().ErrorCode)
		unsupported := p.BooleanEvaluation(context.Background(), "unsupported", defaultValue, nil)
		require.Equal(t, defaultValue, unsupported.Value)
		require.Equal(t, openfeature.ParseErrorCode, unsupported.ResolutionDetail().ErrorCode)
	}
}

func TestUnsupportedDefinitionsAreIsolatedAndReplacePriorValue(t *testing.T) {
	valid := []byte(`{"environment":"production","definitions":{"good":{"variants":[false,true],"environments":{"production":1}},"changed":{"variants":[false,true],"environments":{"production":1}}}}`)
	updated := []byte(`{"environment":"production","definitions":{"good":{"variants":[false,true],"environments":{"production":1}},"changed":{"variants":[false,true],"environments":{"production":{"rules":[{"conditions":[],"outcome":1}],"fallthrough":0}}}}}`)
	p := testProvider(t, valid, updated)
	require.NoError(t, p.refresh(context.Background()))
	require.True(t, p.BooleanEvaluation(context.Background(), "good", false, nil).Value)
	detail := p.BooleanEvaluation(context.Background(), "changed", false, nil)
	require.False(t, detail.Value)
	require.Equal(t, openfeature.ParseErrorCode, detail.ResolutionDetail().ErrorCode)
}

func TestTransportFailureRetainsSnapshotAndRedacts(t *testing.T) {
	body := []byte(`{"environment":"production","definitions":{"flag":{"variants":[false,true],"environments":{"production":1}}}}`)
	p := testProvider(t, body)
	p.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("vf_server_secret target-id") })
	err := p.refresh(context.Background())
	require.NotContains(t, err.Error(), "secret")
	require.NotContains(t, p.Diagnostics().LastRefreshError, "target-id")
	require.True(t, p.BooleanEvaluation(context.Background(), "flag", false, nil).Value)
}

func TestStalenessAndInvalidContext(t *testing.T) {
	body, err := os.ReadFile("testdata/direct-targets.json")
	require.NoError(t, err)
	s, err := parseDatafile(body, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	p := &Provider{config: Config{MaxStaleness: time.Minute}}
	p.snapshot.Store(s)
	stale := p.BooleanEvaluation(context.Background(), "direct", true, nil).ResolutionDetail()
	require.Equal(t, openfeature.ProviderNotReadyCode, stale.ErrorCode)
	require.False(t, p.Diagnostics().Ready)
	s.fetchedAt = time.Now()
	invalid := p.BooleanEvaluation(context.Background(), "direct", true, openfeature.FlattenedContext{"team": "bad"}).ResolutionDetail()
	require.Equal(t, openfeature.InvalidContextCode, invalid.ErrorCode)
}

func TestConcurrentEvaluation(t *testing.T) {
	body, err := os.ReadFile("testdata/direct-targets.json")
	require.NoError(t, err)
	s, err := parseDatafile(body, time.Now())
	require.NoError(t, err)
	p := &Provider{config: Config{MaxStaleness: time.Minute}}
	p.snapshot.Store(s)
	var wait sync.WaitGroup
	results := make(chan openfeature.BoolResolutionDetail, 100)
	for range 100 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results <- p.BooleanEvaluation(context.Background(), "direct", false, openfeature.FlattenedContext{"team": map[string]any{"id": "matching"}})
		}()
	}
	wait.Wait()
	close(results)
	for detail := range results {
		require.True(t, detail.Value, detail.ResolutionDetail().ErrorMessage)
	}
}

func TestInitializationAuthAndShutdown(t *testing.T) {
	body := []byte(`{"environment":"production","definitions":{}}`)
	var authorization string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		authorization = req.Header.Get("Authorization")
		return response(body), nil
	})}
	p, err := newProvider(Config{SDKKey: "vf_server_test", RefreshInterval: time.Second, HTTPTimeout: 0, MaxStaleness: time.Second}, "https://example.invalid/data", client)
	require.NoError(t, err)
	require.NoError(t, p.InitWithContext(context.Background(), openfeature.EvaluationContext{}))
	require.Equal(t, "Bearer vf_server_test", authorization)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)
	require.NoError(t, p.ShutdownWithContext(ctx))
	require.NoError(t, p.ShutdownWithContext(ctx))
}

func TestParseRejectsMalformedAndUnsupportedShapes(t *testing.T) {
	tests := []string{
		`{"variants":[null,true],"environments":{"production":1}}`,
		`{"variants":[true,false],"environments":{"production":null}}`,
		`{"variants":[true,false],"environments":{"production":{"fallthrough":null}}}`,
		`{"variants":[true,false],"environments":{"production":{"targets":[{"team":{"id":[null]}}}],"fallthrough":0}}}`,
		`{"variants":[false,true],"prerequisite":"other","environments":{"production":1}}`,
		`{"variants":[false,true],"environments":{"production":{"fallthrough":2}}}`,
		`{"variants":[false,true],"environments":{"production":{"fallthrough":{"type":"split"}}}}`,
		`{"variants":[false,true],"environments":{"production":{"fallthrough":0,"unknown":true}}}`,
		`{"variants":[false,true],"experiment":{},"environments":{"production":0}}`,
		`{"variants":[false,true],"environments":{"production":{"reuse":"preview"}}}`,
	}
	for _, input := range tests {
		_, err := parseFlag([]byte(input), "production")
		require.Error(t, err)
	}
}

func TestPollingRecoversFromColdFailureThroughOpenFeature(t *testing.T) {
	body, err := os.ReadFile("testdata/direct-targets.json")
	require.NoError(t, err)
	var healthy atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !healthy.Load() {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Authorization") != "Bearer vf_server_test" {
			t.Error("missing SDK authentication")
		}
		if _, err := w.Write(body); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	p, err := newProvider(Config{SDKKey: "vf_server_test", RefreshInterval: time.Second, HTTPTimeout: 0, MaxStaleness: 0}, server.URL, server.Client())
	require.NoError(t, err)
	api, err := featureflag.New(t.Context(), p)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, api.Shutdown(context.Background())) })
	client := api.NewClient()
	evaluation := openfeature.NewTargetlessEvaluationContext(map[string]any{"team": map[string]any{"id": "matching"}})
	detail, err := client.BooleanValueDetails(t.Context(), "direct", false, evaluation)
	require.Error(t, err)
	require.False(t, detail.Value)
	require.Equal(t, "http_401", p.Diagnostics().LastRefreshError)
	require.Positive(t, p.Diagnostics().RefreshFailures)
	healthy.Store(true)
	require.Eventually(t, func() bool {
		value, err := client.BooleanValueDetails(t.Context(), "direct", false, evaluation)
		return err == nil && value.Value
	}, 5*time.Second, 10*time.Millisecond)
	require.True(t, p.Diagnostics().Ready)
	require.Empty(t, p.Diagnostics().LastRefreshError)
	require.NoError(t, api.Shutdown(t.Context()))
	select {
	case <-p.done:
	default:
		t.Fatal("polling continues after shutdown")
	}
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

func TestInvalidContextCannotBeHiddenByAnotherMatchingTarget(t *testing.T) {
	body, err := os.ReadFile("testdata/direct-targets.json")
	require.NoError(t, err)
	p := testProvider(t, body)
	for range 100 {
		detail := p.BooleanEvaluation(t.Context(), "direct", false, openfeature.FlattenedContext{
			"team":    map[string]any{"id": "matching"},
			"account": "invalid",
		})
		require.Equal(t, openfeature.InvalidContextCode, detail.ResolutionDetail().ErrorCode)
	}
}

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
	require.NoError(t, p.refresh(context.Background()))
	return p
}

func response(body []byte) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}
}

func TestNewValidatesConfig(t *testing.T) {
	_, err := New(Config{SDKKey: "vf_server_test", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0})
	require.NoError(t, err)

	for _, config := range []Config{
		{SDKKey: "", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0},
		{SDKKey: "vf_client_test", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0},
		{SDKKey: "vf_server_test", RefreshInterval: time.Millisecond, HTTPTimeout: 0, MaxStaleness: 0},
		{SDKKey: "vf_server_test", RefreshInterval: 0, HTTPTimeout: 2 * time.Minute, MaxStaleness: 0},
		{SDKKey: "vf_server_test", RefreshInterval: time.Minute, HTTPTimeout: 0, MaxStaleness: time.Second},
	} {
		_, err := New(config)
		require.Error(t, err)
	}

	for _, endpoint := range []string{"http://example.invalid/data", "https://user@example.invalid/data", "/data"} {
		_, err := newProvider(Config{SDKKey: "vf_server_test", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0}, endpoint, &http.Client{})
		require.Error(t, err)
	}
}
