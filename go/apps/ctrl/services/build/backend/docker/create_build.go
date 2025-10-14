package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/client"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

type dockerBuildResponse struct {
	Stream      string `json:"stream,omitempty"`
	Error       string `json:"error,omitempty"`
	ErrorDetail struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errorDetail"`
}

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

	d.logger.Info("Getting presigned URL for build context",
		"context_key", req.Msg.ContextKey,
		"unkey_project_id", req.Msg.UnkeyProjectID)

	contextURL, err := d.storage.GetPresignedURL(ctx, req.Msg.ContextKey, 15*time.Minute)
	if err != nil {
		d.logger.Error("Failed to get presigned URL",
			"error", err,
			"context_key", req.Msg.ContextKey,
			"unkey_project_id", req.Msg.UnkeyProjectID)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to get presigned URL: %w", err))
	}

	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		d.logger.Error("Failed to create docker client",
			"error", err,
			"unkey_project_id", req.Msg.UnkeyProjectID)
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

	d.logger.Info("Starting Docker build",
		"image_tag", imageTag,
		"dockerfile", dockerfilePath,
		"platform", "linux/arm64",
		"unkey_project_id", req.Msg.UnkeyProjectID)

	buildOptions := build.ImageBuildOptions{
		Tags:          []string{imageTag},
		Dockerfile:    dockerfilePath,
		Platform:      "linux/arm64",
		Remove:        true,
		RemoteContext: contextURL,
	}

	buildResponse, err := dockerClient.ImageBuild(ctx, nil, buildOptions)
	if err != nil {
		d.logger.Error("Docker build failed",
			"error", err,
			"image_tag", imageTag,
			"dockerfile", dockerfilePath,
			"unkey_project_id", req.Msg.UnkeyProjectID)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to start build: %w", err))
	}
	defer buildResponse.Body.Close()

	scanner := bufio.NewScanner(buildResponse.Body)
	var buildError error

	for scanner.Scan() {
		var resp dockerBuildResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			continue
		}

		if resp.Error != "" {
			buildError = fmt.Errorf("%s", resp.ErrorDetail.Message)
			d.logger.Error("Build failed",
				"error", resp.ErrorDetail.Message,
				"image_tag", imageTag,
				"unkey_project_id", req.Msg.UnkeyProjectID)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		d.logger.Error("Failed to read build output",
			"error", err,
			"image_tag", imageTag,
			"unkey_project_id", req.Msg.UnkeyProjectID)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to read build output: %w", err))
	}

	if buildError != nil {
		return nil, connect.NewError(connect.CodeInternal, buildError)
	}

	buildID := fmt.Sprintf("docker-%d", timestamp)

	d.logger.Info("Build completed successfully",
		"image_tag", imageTag,
		"build_id", buildID,
		"unkey_project_id", req.Msg.UnkeyProjectID)

	return connect.NewResponse(&ctrlv1.CreateBuildResponse{
		ImageName: imageTag,
		BuildId:   buildID,
	}), nil
}
