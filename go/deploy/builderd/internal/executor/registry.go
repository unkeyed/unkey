package executor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/observability"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1"
)

// Registry manages different build executors
type Registry struct {
	logger    *slog.Logger
	config    *config.Config
	executors map[string]Executor
	mutex     sync.RWMutex
}

// NewRegistry creates a new executor registry
func NewRegistry(logger *slog.Logger, cfg *config.Config, buildMetrics *observability.BuildMetrics) *Registry {
	registry := &Registry{ //nolint:exhaustruct // mutex is zero-value initialized and doesn't need explicit initialization
		logger:    logger,
		config:    cfg,
		executors: make(map[string]Executor),
	}

	// Register built-in executors
	registry.registerBuiltinExecutors(buildMetrics)

	return registry
}

// registerBuiltinExecutors registers the standard executors
func (r *Registry) registerBuiltinExecutors(buildMetrics *observability.BuildMetrics) {
	// Register Docker executor based on feature flag
	if r.config.Builder.UsePipelineExecutor {
		r.logger.InfoContext(context.Background(), "using step-based pipeline executor for Docker builds")
		pipelineExecutor := NewDockerPipelineExecutor(r.logger, r.config, buildMetrics)
		r.RegisterExecutor("docker", pipelineExecutor)
	} else {
		r.logger.InfoContext(context.Background(), "using monolithic executor for Docker builds")
		dockerExecutor := NewDockerExecutor(r.logger, r.config, buildMetrics)
		r.RegisterExecutor("docker", dockerExecutor)
	}

	// TODO: Register other executors
	// gitExecutor := NewGitExecutor(r.logger, r.config, buildMetrics)
	// r.RegisterExecutor("git", gitExecutor)

	// archiveExecutor := NewArchiveExecutor(r.logger, r.config, buildMetrics)
	// r.RegisterExecutor("archive", archiveExecutor)

	r.logger.InfoContext(context.Background(), "registered built-in executors",
		slog.Int("executor_count", len(r.executors)),
		slog.Bool("pipeline_mode", r.config.Builder.UsePipelineExecutor),
	)
}

// RegisterExecutor registers a new executor for a source type
func (r *Registry) RegisterExecutor(sourceType string, executor Executor) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.executors[sourceType] = executor
	r.logger.InfoContext(context.Background(), "registered executor", slog.String("source_type", sourceType))
}

// GetExecutor returns the executor for a given source type
func (r *Registry) GetExecutor(sourceType string) (Executor, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	executor, exists := r.executors[sourceType]
	if !exists {
		r.logger.ErrorContext(context.Background(), "no executor found for source type",
			slog.String("source_type", sourceType),
			slog.Any("available_types", r.GetSupportedSources()),
		)
		return nil, fmt.Errorf("no executor found for source type: %s", sourceType)
	}

	return executor, nil
}

// Execute processes a build request using the appropriate executor
func (r *Registry) Execute(ctx context.Context, request *builderv1.CreateBuildRequest) (*BuildResult, error) {
	// Determine source type from request
	sourceType, err := r.getSourceTypeFromRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to determine source type: %w", err)
	}

	// Get appropriate executor
	executor, err := r.GetExecutor(sourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor: %w", err)
	}

	r.logger.InfoContext(ctx, "executing build request",
		slog.String("source_type", sourceType),
	)

	// Execute the build
	result, err := executor.Execute(ctx, request)
	if err != nil {
		r.logger.ErrorContext(ctx, "build execution failed",
			slog.String("source_type", sourceType),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("build execution failed: %w", err)
	}

	r.logger.InfoContext(ctx, "build execution completed",
		slog.String("source_type", sourceType),
		slog.String("build_id", result.BuildID),
		slog.String("status", result.Status),
	)

	return result, nil
}

// ExecuteWithID processes a build request with a pre-assigned build ID
func (r *Registry) ExecuteWithID(ctx context.Context, request *builderv1.CreateBuildRequest, buildID string) (*BuildResult, error) {
	// Determine source type from request
	sourceType, err := r.getSourceTypeFromRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to determine source type: %w", err)
	}

	// Get appropriate executor
	executor, err := r.GetExecutor(sourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor: %w", err)
	}

	r.logger.InfoContext(ctx, "executing build request with ID",
		slog.String("source_type", sourceType),
		slog.String("build_id", buildID),
	)

	// Execute the build with the provided ID
	result, err := executor.ExecuteWithID(ctx, request, buildID)
	if err != nil {
		r.logger.ErrorContext(ctx, "build execution failed",
			slog.String("source_type", sourceType),
			slog.String("build_id", buildID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("build execution failed: %w", err)
	}

	r.logger.InfoContext(ctx, "build execution completed",
		slog.String("source_type", sourceType),
		slog.String("build_id", result.BuildID),
		slog.String("status", result.Status),
	)

	return result, nil
}

// getSourceTypeFromRequest determines the source type from the build request
func (r *Registry) getSourceTypeFromRequest(request *builderv1.CreateBuildRequest) (string, error) {
	if request.GetConfig() == nil || request.GetConfig().GetSource() == nil {
		r.logger.ErrorContext(context.Background(), "build source is required but missing")
		return "", fmt.Errorf("build source is required")
	}

	switch source := request.GetConfig().GetSource().GetSourceType().(type) {
	case *builderv1.BuildSource_DockerImage:
		return "docker", nil
	case *builderv1.BuildSource_GitRepository:
		return "git", nil
	case *builderv1.BuildSource_Archive:
		return "archive", nil
	default:
		r.logger.ErrorContext(context.Background(), "unsupported source type",
			slog.String("type", fmt.Sprintf("%T", source)),
		)
		return "", fmt.Errorf("unsupported source type: %T", source)
	}
}

// ListExecutors returns a list of registered executors
func (r *Registry) ListExecutors() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	executors := make([]string, 0, len(r.executors))
	for sourceType := range r.executors {
		executors = append(executors, sourceType)
	}

	return executors
}

// Cleanup removes temporary resources for all executors
func (r *Registry) Cleanup(ctx context.Context, buildID string) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var lastError error

	for sourceType, executor := range r.executors {
		if err := executor.Cleanup(ctx, buildID); err != nil {
			r.logger.WarnContext(ctx, "executor cleanup failed",
				slog.String("source_type", sourceType),
				slog.String("build_id", buildID),
				slog.String("error", err.Error()),
			)
			lastError = err
		}
	}

	return lastError
}

// GetSupportedSources returns all supported source types
func (r *Registry) GetSupportedSources() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	sources := make([]string, 0, len(r.executors))
	for sourceType := range r.executors {
		sources = append(sources, sourceType)
	}

	return sources
}
