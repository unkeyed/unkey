package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/sentinel/services/router"
)

const (
	testPlatform = "test"
	testRegion   = "test-region"
)

// setupHandler creates a real router service backed by MySQL and returns a
// Handler wired to the given upstream address.
func setupHandler(t *testing.T, upstreamAddr string) (*Handler, string) {
	t.Helper()

	mysqlCfg := dockertest.MySQL(t)
	clk := clock.New()

	database, err := db.New(db.Config{
		PrimaryDSN:  mysqlCfg.DSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	ctx := context.Background()
	s := seed.New(t, database, nil)
	ws := s.CreateWorkspace(ctx)

	project := s.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New("prj"),
		WorkspaceID: ws.ID,
		Name:        "test-project",
		Slug:        uid.New("slug"),
	})

	app := s.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		Name:          "default",
		Slug:          "default",
		DefaultBranch: "main",
	})

	env := s.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New("env"),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
	})

	deployment := s.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        db.DeploymentsStatusReady,
	})

	region := s.CreateRegion(ctx, seed.CreateRegionRequest{
		Name:     testRegion,
		Platform: testPlatform,
	})

	s.CreateInstance(ctx, seed.CreateInstanceRequest{
		DeploymentID: deployment.ID,
		WorkspaceID:  ws.ID,
		ProjectID:    project.ID,
		AppID:        app.ID,
		RegionID:     region.ID,
		Address:      upstreamAddr,
	})

	routerSvc, err := router.New(router.Config{
		DB:            database,
		Clock:         clk,
		EnvironmentID: env.ID,
		Platform:      testPlatform,
		Region:        testRegion,
		Broadcaster:   nil,
		NodeID:        "test-node",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routerSvc.Close() })

	// nolint:exhaustruct
	h := &Handler{
		RouterService: routerSvc,
		Clock:         clk,
		Transports:    NewTransportRegistry(),
		SentinelID:    "sentinel_test",
		Region:        testRegion,
		Engine:        nil,
	}

	return h, deployment.ID
}

// proxyStreaming sends a streaming (gRPC content-type) request through the
// handler and returns the tracking result.
func proxyStreaming(t *testing.T, h *Handler, deploymentID string, requestBody string) (*SentinelRequestTracking, []byte) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/some/path", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/grpc")
	req.Header.Set("X-Deployment-Id", deploymentID)

	w := httptest.NewRecorder()
	sess := &zen.Session{}
	err := sess.Init(w, req, 0)
	require.NoError(t, err)

	tracking := &SentinelRequestTracking{
		StartTime: time.Now(),
	}
	ctx := WithSentinelTracking(context.Background(), tracking)

	err = h.Handle(ctx, sess)
	require.NoError(t, err)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return tracking, body
}

func TestHandler_StreamingBodyCapture(t *testing.T) {
	chunks := []string{"chunk1\n", "chunk2\n", "chunk3\n"}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		w.Header().Set("Content-Type", "application/grpc")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		for _, chunk := range chunks {
			_, _ = fmt.Fprint(w, chunk)
			flusher.Flush()
		}
		_, _ = fmt.Fprintf(w, "req:%s", reqBody)
		flusher.Flush()
	}))
	defer upstream.Close()

	h, deploymentID := setupHandler(t, strings.TrimPrefix(upstream.URL, "http://"))

	requestBody := "streaming request payload"
	tracking, respBody := proxyStreaming(t, h, deploymentID, requestBody)

	for _, chunk := range chunks {
		require.Contains(t, string(respBody), chunk, "client should have received chunk")
	}
	require.Contains(t, string(respBody), "req:"+requestBody, "upstream should have received the request body")

	require.NotEmpty(t, tracking.RequestBody, "streaming request body should be captured")
	require.Equal(t, requestBody, string(tracking.RequestBody))

	require.NotEmpty(t, tracking.ResponseBody, "streaming response body should be captured")
	require.Contains(t, string(tracking.ResponseBody), "chunk1")
	require.Contains(t, string(tracking.ResponseBody), "chunk3")
}

// TestHandler_ServerStreamBodyCapture verifies that a non-streaming request
// with a streaming *response* (server-stream) still captures the response body
// via TeeReader based on the response Content-Type.
func TestHandler_ServerStreamBodyCapture(t *testing.T) {
	chunks := []string{"chunk1\n", "chunk2\n", "chunk3\n"}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()

		// Response is streaming (Connect), but request was application/json
		w.Header().Set("Content-Type", "application/connect+proto")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		for _, chunk := range chunks {
			_, _ = fmt.Fprint(w, chunk)
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	h, deploymentID := setupHandler(t, strings.TrimPrefix(upstream.URL, "http://"))

	// Send a normal JSON request — NOT application/grpc
	req := httptest.NewRequest(http.MethodPost, "/some/path", strings.NewReader(`{"msg":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Deployment-Id", deploymentID)

	w := httptest.NewRecorder()
	sess := &zen.Session{}
	err := sess.Init(w, req, 0)
	require.NoError(t, err)

	tracking := &SentinelRequestTracking{StartTime: time.Now()}
	ctx := WithSentinelTracking(context.Background(), tracking)

	err = h.Handle(ctx, sess)
	require.NoError(t, err)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)

	for _, chunk := range chunks {
		require.Contains(t, string(respBody), chunk, "client should have received chunk")
	}

	require.NotEmpty(t, tracking.ResponseBody, "server-stream response body should be captured via TeeReader")
	require.Contains(t, string(tracking.ResponseBody), "chunk1")
	require.Contains(t, string(tracking.ResponseBody), "chunk3")
}

func TestHandler_StreamingBodyCapture_RespectsLimit(t *testing.T) {
	bigChunk := strings.Repeat("X", zen.MaxBodyCapture+1024)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Drain the request body so the TeeReader captures it.
		_, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()
		w.Header().Set("Content-Type", "application/grpc")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, bigChunk)
	}))
	defer upstream.Close()

	h, deploymentID := setupHandler(t, strings.TrimPrefix(upstream.URL, "http://"))

	tracking, respBody := proxyStreaming(t, h, deploymentID, bigChunk)

	require.Len(t, respBody, len(bigChunk), "client must receive the full response even though logging is capped")
	require.Len(t, tracking.RequestBody, zen.MaxBodyCapture, "request body capture should be capped")
	require.Len(t, tracking.ResponseBody, zen.MaxBodyCapture, "response body capture should be capped")
}
