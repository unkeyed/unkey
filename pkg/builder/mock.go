package builder

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// BuildStatus represents the status of a build in the builder service
type BuildStatus string

const (
	BuildStatusQueued  BuildStatus = "queued"
	BuildStatusRunning BuildStatus = "running"
	BuildStatusSuccess BuildStatus = "success"
	BuildStatusFailed  BuildStatus = "failed"
)

// BuildInfo represents build information from the builder service
type BuildInfo struct {
	BuildID     string      `json:"build_id"`
	DockerImage string      `json:"docker_image"`
	Status      BuildStatus `json:"status"`
	ErrorMsg    string      `json:"error_msg,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

// Service defines the interface for the builder service
type Service interface {
	// SubmitBuild submits a new build to the builder service
	SubmitBuild(ctx context.Context, buildID, dockerImage string) error

	// GetBuildStatus gets the current status of a build
	GetBuildStatus(ctx context.Context, buildID string) (*BuildInfo, error)
}

// MockService implements a mock builder service for testing
type MockService struct {
	mu     sync.RWMutex
	builds map[string]*BuildInfo
}

// NewMockService creates a new mock builder service
func NewMockService() *MockService {
	return &MockService{
		mu:     sync.RWMutex{},
		builds: make(map[string]*BuildInfo),
	}
}

// SubmitBuild submits a build to the mock service
func (m *MockService) SubmitBuild(ctx context.Context, buildID, dockerImage string) error {
	now := time.Now()
	m.mu.Lock()
	m.builds[buildID] = &BuildInfo{
		BuildID:     buildID,
		DockerImage: dockerImage,
		Status:      BuildStatusQueued,
		ErrorMsg:    "",
		CreatedAt:   now,
		StartedAt:   nil,
		CompletedAt: nil,
	}
	m.mu.Unlock()

	// Simulate async processing by starting a goroutine that updates status
	go m.simulateBuild(buildID)

	return nil
}

// GetBuildStatus gets the status of a build
func (m *MockService) GetBuildStatus(ctx context.Context, buildID string) (*BuildInfo, error) {
	m.mu.RLock()
	build, exists := m.builds[buildID]
	if !exists {
		m.mu.RUnlock()
		return nil, fmt.Errorf("build %s not found", buildID)
	}

	// Return a copy to avoid race conditions
	buildCopy := *build
	m.mu.RUnlock()
	return &buildCopy, nil
}

// simulateBuild simulates the build process with realistic timing
func (m *MockService) simulateBuild(buildID string) {
	m.mu.RLock()
	_, exists := m.builds[buildID]
	m.mu.RUnlock()
	if !exists {
		return
	}

	// Queue for 1-3 seconds
	time.Sleep(time.Duration(1+rand.Intn(3)) * time.Second) // nolint:gosec // Weak random is acceptable for mock simulation

	// Start building
	now := time.Now()
	m.mu.Lock()
	if build, ok := m.builds[buildID]; ok {
		build.Status = BuildStatusRunning
		build.StartedAt = &now
	}
	m.mu.Unlock()

	// Build for 8-15 seconds
	buildDuration := time.Duration(8+rand.Intn(8)) * time.Second // nolint:gosec // Weak random is acceptable for mock simulation
	time.Sleep(buildDuration)

	// Complete (90% success rate)
	completedAt := time.Now()
	m.mu.Lock()
	if build, ok := m.builds[buildID]; ok {
		build.CompletedAt = &completedAt
		if rand.Float32() < 0.9 { // nolint:gosec // Weak random is acceptable for mock simulation
			build.Status = BuildStatusSuccess
		} else {
			build.Status = BuildStatusFailed
			build.ErrorMsg = "Mock build failure: Docker build failed with exit code 1"
		}
	}
	m.mu.Unlock()
}
