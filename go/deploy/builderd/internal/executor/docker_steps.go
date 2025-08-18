package executor

import (
	"context"
	"log/slog"
)

// PullImageStep pulls the Docker image
type PullImageStep struct {
	executor *DockerExecutor
}

func (s *PullImageStep) Name() string {
	return "pull_image"
}

func (s *PullImageStep) Execute(ctx context.Context, input StepInput) (StepOutput, error) {
	imageName := input.Config.GetSource().GetDockerImage().GetImageUri()

	input.Logger.InfoContext(ctx, "pulling Docker image", slog.String("image", imageName))

	err := s.executor.pullDockerImage(ctx, input.Logger, imageName)
	if err != nil {
		return StepOutput{Success: false, Error: err}, err
	}

	return StepOutput{
		ImageName: imageName,
		Success:   true,
	}, nil
}

// CreateContainerStep creates a container from the image
type CreateContainerStep struct {
	executor *DockerExecutor
}

func (s *CreateContainerStep) Name() string {
	return "create_container"
}

func (s *CreateContainerStep) Execute(ctx context.Context, input StepInput) (StepOutput, error) {
	input.Logger.InfoContext(ctx, "creating container", slog.String("image", input.ImageName))

	containerID, err := s.executor.createContainer(ctx, input.Logger, input.ImageName)
	if err != nil {
		return StepOutput{Success: false, Error: err}, err
	}

	return StepOutput{
		ImageName:   input.ImageName,
		ContainerID: containerID,
		Success:     true,
	}, nil
}

// ExtractMetadataStep extracts container metadata
type ExtractMetadataStep struct {
	executor *DockerExecutor
}

func (s *ExtractMetadataStep) Name() string {
	return "extract_metadata"
}

func (s *ExtractMetadataStep) Execute(ctx context.Context, input StepInput) (StepOutput, error) {
	input.Logger.InfoContext(ctx, "extracting container metadata", slog.String("image", input.ImageName))

	metadata, err := s.executor.extractContainerMetadata(ctx, input.Logger, input.ImageName)
	if err != nil {
		return StepOutput{Success: false, Error: err}, err
	}

	return StepOutput{
		ImageName:   input.ImageName,
		ContainerID: input.ContainerID,
		Metadata:    metadata,
		Success:     true,
	}, nil
}

// ExtractFilesystemStep extracts the container filesystem
type ExtractFilesystemStep struct {
	executor *DockerExecutor
}

func (s *ExtractFilesystemStep) Name() string {
	return "extract_filesystem"
}

func (s *ExtractFilesystemStep) Execute(ctx context.Context, input StepInput) (StepOutput, error) {
	input.Logger.InfoContext(ctx, "extracting container filesystem",
		slog.String("container_id", input.ContainerID),
		slog.String("rootfs_dir", input.RootfsDir))

	err := s.executor.extractFilesystem(ctx, input.Logger, input.ContainerID, input.RootfsDir, input.Metadata)
	if err != nil {
		return StepOutput{Success: false, Error: err}, err
	}

	return StepOutput{
		ImageName:   input.ImageName,
		ContainerID: input.ContainerID,
		Metadata:    input.Metadata,
		RootfsPath:  input.RootfsDir,
		Success:     true,
	}, nil
}

// OptimizeRootfsStep optimizes the extracted rootfs
type OptimizeRootfsStep struct {
	executor *DockerExecutor
}

func (s *OptimizeRootfsStep) Name() string {
	return "optimize_rootfs"
}

func (s *OptimizeRootfsStep) Execute(ctx context.Context, input StepInput) (StepOutput, error) {
	input.Logger.InfoContext(ctx, "optimizing rootfs", slog.String("rootfs_dir", input.RootfsDir))

	// Call existing optimization logic from DockerExecutor
	// This would include: creating metald-init, container command/env files, etc.
	err := s.executor.injectMetaldInit(ctx, input.Logger, input.RootfsDir)
	if err != nil {
		return StepOutput{Success: false, Error: err}, err
	}

	err = s.executor.createContainerCmd(ctx, input.Logger, input.RootfsDir, input.Metadata)
	if err != nil {
		return StepOutput{Success: false, Error: err}, err
	}

	err = s.executor.createContainerEnv(ctx, input.Logger, input.RootfsDir, input.Metadata)
	if err != nil {
		return StepOutput{Success: false, Error: err}, err
	}

	return StepOutput{
		ImageName:   input.ImageName,
		ContainerID: input.ContainerID,
		Metadata:    input.Metadata,
		RootfsPath:  input.RootfsDir,
		Success:     true,
	}, nil
}

// CleanupStep cleans up temporary resources
type CleanupStep struct {
	executor *DockerExecutor
}

func (s *CleanupStep) Name() string {
	return "cleanup"
}

func (s *CleanupStep) Execute(ctx context.Context, input StepInput) (StepOutput, error) {
	input.Logger.InfoContext(ctx, "cleaning up container", slog.String("container_id", input.ContainerID))

	if input.ContainerID != "" {
		err := s.executor.removeContainer(ctx, input.Logger, input.ContainerID)
		if err != nil {
			// Log warning but don't fail the build for cleanup errors
			input.Logger.WarnContext(ctx, "failed to cleanup container", slog.String("error", err.Error()))
		}
	}

	return StepOutput{
		ImageName:   input.ImageName,
		ContainerID: "", // Container is now removed
		Metadata:    input.Metadata,
		RootfsPath:  input.RootfsDir,
		Success:     true,
	}, nil
}
