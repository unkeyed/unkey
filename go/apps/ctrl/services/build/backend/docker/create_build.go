package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/client"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (d *Docker) CreateBuild(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateBuildRequest],
) (*connect.Response[ctrlv1.CreateBuildResponse], error) {
	if req.Msg.ContextKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("contextKey is required"))
	}
	if req.Msg.UnkeyProjectID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unkeyProjectID is required"))
	}

	// Download tar from S3
	d.logger.Info("Downloading build context from S3", "context_key", req.Msg.ContextKey)
	tarData, exists, err := d.storage.GetObject(ctx, req.Msg.ContextKey)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to download build context: %w", err))
	}
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("build context not found: %s", req.Msg.ContextKey))
	}
	d.logger.Info("Build context downloaded", "size_bytes", len(tarData))

	// Create Docker client
	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to create docker client: %w", err))
	}
	defer dockerClient.Close()

	timestamp := time.Now().UnixMilli()
	imageTag := fmt.Sprintf("%s:%d",
		strings.ToLower(req.Msg.UnkeyProjectID),
		timestamp,
	)

	dockerfilePath := req.Msg.GetDockerfilePath()
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	d.logger.Info("Starting Docker build", "image_tag", imageTag, "dockerfile", dockerfilePath)

	buildOptions := build.ImageBuildOptions{
		Tags:       []string{imageTag},
		Dockerfile: dockerfilePath,
		Platform:   "linux/arm64",
		Remove:     true,
	}

	// Build image with tar from memory
	buildResponse, err := dockerClient.ImageBuild(
		ctx,
		bytes.NewReader(tarData),
		buildOptions,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to start build: %w", err))
	}
	defer buildResponse.Body.Close()

	// Stream build logs
	if _, err := io.Copy(io.Discard, buildResponse.Body); err != nil {
		d.logger.Error("Failed to read build output", "error", err)
	}

	d.logger.Info("Build completed successfully", "image_tag", imageTag)

	buildID := fmt.Sprintf("docker-%d", timestamp)
	return connect.NewResponse(&ctrlv1.CreateBuildResponse{
		ImageName: imageTag,
		BuildId:   buildID,
	}), nil
}
