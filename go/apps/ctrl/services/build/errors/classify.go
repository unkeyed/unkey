package errors

import (
	"fmt"
	"strings"

	"connectrpc.com/connect"
)

// ClassifyBuildError analyzes build errors and returns appropriate error codes and messages
func ClassifyBuildError(buildError error, dockerfilePath string) error {
	errorMsg := buildError.Error()

	// Check for Dockerfile-related errors
	if strings.Contains(errorMsg, "failed to solve with frontend dockerfile.v0") ||
		strings.Contains(errorMsg, "failed to read dockerfile") ||
		strings.Contains(errorMsg, "failed to solve: failed to read dockerfile") ||
		strings.Contains(errorMsg, "no such file or directory") && strings.Contains(errorMsg, dockerfilePath) {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("dockerfile not found: the file '%s' does not exist in the build context. Please check the dockerfile path and ensure it exists", dockerfilePath))
	}

	// Check for permission errors
	if strings.Contains(errorMsg, "permission denied") {
		return connect.NewError(connect.CodePermissionDenied,
			fmt.Errorf("permission denied: unable to access dockerfile or build context. Please check file permissions"))
	}

	// Default to internal error for other build failures
	return connect.NewError(connect.CodeInternal,
		fmt.Errorf("build failed: %w", buildError))
}
