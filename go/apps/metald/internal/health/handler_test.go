package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
)

// mockBackend is a mock implementation of types.Backend for testing
type mockBackend struct {
	mock.Mock
}

func (m *mockBackend) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
	args := m.Called(ctx, config)
	return args.String(0), args.Error(1)
}

func (m *mockBackend) DeleteVM(ctx context.Context, vmID string) error {
	args := m.Called(ctx, vmID)
	return args.Error(0)
}

func (m *mockBackend) BootVM(ctx context.Context, vmID string) error {
	args := m.Called(ctx, vmID)
	return args.Error(0)
}

func (m *mockBackend) ShutdownVM(ctx context.Context, vmID string) error {
	args := m.Called(ctx, vmID)
	return args.Error(0)
}

func (m *mockBackend) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error {
	args := m.Called(ctx, vmID, force, timeoutSeconds)
	return args.Error(0)
}

func (m *mockBackend) PauseVM(ctx context.Context, vmID string) error {
	args := m.Called(ctx, vmID)
	return args.Error(0)
}

func (m *mockBackend) ResumeVM(ctx context.Context, vmID string) error {
	args := m.Called(ctx, vmID)
	return args.Error(0)
}

func (m *mockBackend) RebootVM(ctx context.Context, vmID string) error {
	args := m.Called(ctx, vmID)
	return args.Error(0)
}

func (m *mockBackend) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	args := m.Called(ctx, vmID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.VMInfo), args.Error(1)
}

func (m *mockBackend) GetVMMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) {
	args := m.Called(ctx, vmID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.VMMetrics), args.Error(1)
}

func (m *mockBackend) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewHandler(t *testing.T) {
	backend := &mockBackend{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	startTime := time.Now()

	handler := NewHandler(backend, logger, startTime)

	assert.NotNil(t, handler)
	assert.Equal(t, backend, handler.backend)
	assert.Equal(t, startTime, handler.startTime)
	assert.NotNil(t, handler.logger)
}

func TestHandler_ServeHTTP_Healthy(t *testing.T) {
	backend := &mockBackend{}
	backend.On("Ping", mock.Anything).Return(nil)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	handler := NewHandler(backend, logger, time.Now())

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))

	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, StatusHealthy, response.Status)
	assert.Equal(t, "dev", response.Version)
	assert.Equal(t, "firecracker", response.Backend.Type)
	assert.Equal(t, StatusHealthy, response.Backend.Status)
	assert.Empty(t, response.Backend.Error)
	assert.Contains(t, response.Checks, "backend_ping")
	assert.Contains(t, response.Checks, "system_info")

	backend.AssertExpectations(t)
}

func TestHandler_ServeHTTP_Unhealthy(t *testing.T) {
	backend := &mockBackend{}
	backendErr := errors.New("backend unavailable")
	backend.On("Ping", mock.Anything).Return(backendErr)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	handler := NewHandler(backend, logger, time.Now())

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// The status will be "degraded" because backend failure creates a check that fails
	// which overrides the backend unhealthy status with degraded
	assert.Equal(t, StatusDegraded, response.Status)
	assert.Equal(t, "firecracker", response.Backend.Type)
	assert.Equal(t, StatusUnhealthy, response.Backend.Status)
	assert.Equal(t, "backend unavailable", response.Backend.Error)

	backend.AssertExpectations(t)
}

func TestHandler_performHealthChecks(t *testing.T) {
	tests := []struct {
		name           string
		backendPingErr error
		expectedStatus string
		expectedChecks int
	}{
		{
			name:           "all healthy",
			backendPingErr: nil,
			expectedStatus: StatusHealthy,
			expectedChecks: 2, // backend_ping + system_info
		},
		{
			name:           "backend unhealthy",
			backendPingErr: errors.New("ping failed"),
			expectedStatus: StatusDegraded, // Backend failure creates unhealthy check which sets status to degraded
			expectedChecks: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &mockBackend{}
			backend.On("Ping", mock.Anything).Return(tt.backendPingErr)

			logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
			handler := NewHandler(backend, logger, time.Now())

			ctx := context.Background()
			response := handler.performHealthChecks(ctx)

			assert.Equal(t, tt.expectedStatus, response.Status)
			assert.Equal(t, "dev", response.Version)
			assert.Equal(t, "firecracker", response.Backend.Type)
			assert.Len(t, response.Checks, tt.expectedChecks)
			assert.Contains(t, response.Checks, "backend_ping")
			assert.Contains(t, response.Checks, "system_info")

			backend.AssertExpectations(t)
		})
	}
}

func TestHandler_checkBackendHealth(t *testing.T) {
	tests := []struct {
		name           string
		backendPingErr error
		expectedStatus string
		expectError    bool
	}{
		{
			name:           "backend healthy",
			backendPingErr: nil,
			expectedStatus: StatusHealthy,
			expectError:    false,
		},
		{
			name:           "backend unhealthy",
			backendPingErr: errors.New("connection failed"),
			expectedStatus: StatusUnhealthy,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &mockBackend{}
			backend.On("Ping", mock.Anything).Return(tt.backendPingErr)

			logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
			handler := NewHandler(backend, logger, time.Now())

			ctx := context.Background()
			checks := make(map[string]Check)

			backendHealth := handler.checkBackendHealth(ctx, checks)

			assert.Equal(t, "firecracker", backendHealth.Type)
			assert.Equal(t, tt.expectedStatus, backendHealth.Status)

			if tt.expectError {
				assert.NotEmpty(t, backendHealth.Error)
				assert.Equal(t, tt.backendPingErr.Error(), backendHealth.Error)
			} else {
				assert.Empty(t, backendHealth.Error)
			}

			// Verify check was added
			require.Contains(t, checks, "backend_ping")
			pingCheck := checks["backend_ping"]
			assert.Equal(t, tt.expectedStatus, pingCheck.Status)

			if tt.expectError {
				assert.NotEmpty(t, pingCheck.Error)
			} else {
				assert.Empty(t, pingCheck.Error)
			}

			backend.AssertExpectations(t)
		})
	}
}

func TestHandler_checkBackendHealth_Timeout(t *testing.T) {
	backend := &mockBackend{}

	// Mock returns context deadline exceeded error
	timeoutErr := context.DeadlineExceeded
	backend.On("Ping", mock.Anything).Return(timeoutErr)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	handler := NewHandler(backend, logger, time.Now())

	ctx := context.Background()
	checks := make(map[string]Check)

	backendHealth := handler.checkBackendHealth(ctx, checks)

	assert.Equal(t, "firecracker", backendHealth.Type)
	assert.Equal(t, StatusUnhealthy, backendHealth.Status)
	assert.Contains(t, backendHealth.Error, "context deadline exceeded")

	require.Contains(t, checks, "backend_ping")
	pingCheck := checks["backend_ping"]
	assert.Equal(t, StatusUnhealthy, pingCheck.Status)
	assert.Contains(t, pingCheck.Error, "context deadline exceeded")

	backend.AssertExpectations(t)
}

func TestHealthResponse_JSON(t *testing.T) {
	response := &HealthResponse{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Backend: BackendHealth{
			Type:   "firecracker",
			Status: StatusHealthy,
		},
		Checks: map[string]Check{
			"backend_ping": {
				Status:    StatusHealthy,
				Duration:  100 * time.Millisecond,
				Timestamp: time.Now(),
			},
		},
	}

	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	var unmarshaled HealthResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response.Status, unmarshaled.Status)
	assert.Equal(t, response.Version, unmarshaled.Version)
	assert.Equal(t, response.Backend.Type, unmarshaled.Backend.Type)
	assert.Equal(t, response.Backend.Status, unmarshaled.Backend.Status)
	assert.Contains(t, unmarshaled.Checks, "backend_ping")
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, "healthy", StatusHealthy)
	assert.Equal(t, "unhealthy", StatusUnhealthy)
	assert.Equal(t, "degraded", StatusDegraded)
}
