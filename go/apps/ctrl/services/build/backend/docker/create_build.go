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
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/uid"
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
	buildContextPath := req.Msg.GetBuildContextPath()
	unkeyProjectID := req.Msg.GetUnkeyProjectId()
	deploymentID := req.Msg.GetDeploymentId()

	if err := assert.All(
		assert.NotEmpty(buildContextPath, "build_context_path is required"),
		assert.NotEmpty(unkeyProjectID, "unkey_project_id is required"),
		assert.NotEmpty(deploymentID, "deploymentID is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Use configured platform from config
	platform := d.buildPlatform.Platform
	architecture := d.buildPlatform.Architecture

	d.logger.Info("Getting presigned URL for build context",
		"build_context_path", buildContextPath,
		"unkey_project_id", unkeyProjectID,
		"platform", platform,
		"architecture", architecture)

	contextURL, err := d.storage.GenerateDownloadURL(ctx, buildContextPath, 15*time.Minute)
	if err != nil {
		d.logger.Error("Failed to get presigned URL",
			"error", err,
			"build_context_path", buildContextPath,
			"unkey_project_id", unkeyProjectID)
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
			"unkey_project_id", unkeyProjectID)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to create docker client: %w", err))
	}
	defer dockerClient.Close()

	// Docker requires lowercase repository names
	imageName := strings.ToLower(fmt.Sprintf("%s-%s",
		unkeyProjectID,
		deploymentID,
	))

	dockerfilePath := req.Msg.GetDockerfilePath()
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	d.logger.Info("Starting Docker build",
		"image_name", imageName,
		"dockerfile", dockerfilePath,
		"platform", platform,
		"architecture", architecture,
		"unkey_project_id", unkeyProjectID)

	buildOptions := build.ImageBuildOptions{
		Tags:          []string{imageName},
		Dockerfile:    dockerfilePath,
		Platform:      platform,
		Remove:        true,
		RemoteContext: contextURL,
	}

	buildResponse, err := dockerClient.ImageBuild(ctx, nil, buildOptions)
	if err != nil {
		d.logger.Error("Docker build failed",
			"error", err,
			"image_name", imageName,
			"dockerfile", dockerfilePath,
			"unkey_project_id", unkeyProjectID)
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
				"image_name", imageName,
				"unkey_project_id", unkeyProjectID)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		d.logger.Error("Failed to read build output",
			"error", err,
			"image_name", imageName,
			"unkey_project_id", unkeyProjectID)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to read build output: %w", err))
	}

	if buildError != nil {
		return nil, connect.NewError(connect.CodeInternal, buildError)
	}

	buildID := uid.New(uid.BuildPrefix)

	d.logger.Info("Build completed successfully",
		"image_name", imageName,
		"build_id", buildID,
		"platform", platform,
		"architecture", architecture,
		"unkey_project_id", unkeyProjectID)

	return connect.NewResponse(&ctrlv1.CreateBuildResponse{
		ImageName: imageName,
		BuildId:   buildID,
	}), nil
}
