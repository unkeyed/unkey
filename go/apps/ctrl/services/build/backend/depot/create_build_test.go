package depot

import (
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
)

func TestClassifyBuildError(t *testing.T) {
	tests := []struct {
		name           string
		buildError     error
		dockerfilePath string
		expectedCode   connect.Code
		shouldContain  []string
	}{
		{
			name:           "dockerfile not found error - buildkit",
			buildError:     errors.New("failed to solve: failed to read dockerfile: open Dockerfile: no such file or directory"),
			dockerfilePath: "Dockerfile",
			expectedCode:   connect.CodeInvalidArgument,
			shouldContain: []string{
				"dockerfile not found",
				"Dockerfile",
				"does not exist",
			},
		},
		{
			name:           "dockerfile not found with frontend error",
			buildError:     errors.New("failed to solve with frontend dockerfile.v0: failed to read dockerfile: open Dockerfile.prod: no such file or directory"),
			dockerfilePath: "Dockerfile.prod",
			expectedCode:   connect.CodeInvalidArgument,
			shouldContain: []string{
				"dockerfile not found",
				"Dockerfile.prod",
			},
		},
		{
			name:           "custom dockerfile path not found",
			buildError:     errors.New("no such file or directory: docker/Dockerfile"),
			dockerfilePath: "docker/Dockerfile",
			expectedCode:   connect.CodeInvalidArgument,
			shouldContain: []string{
				"dockerfile not found",
				"docker/Dockerfile",
			},
		},
		{
			name:           "permission denied error",
			buildError:     errors.New("permission denied: cannot read dockerfile"),
			dockerfilePath: "Dockerfile",
			expectedCode:   connect.CodePermissionDenied,
			shouldContain: []string{
				"permission denied",
			},
		},
		{
			name:           "generic build error",
			buildError:     errors.New("build failed: image pull failed"),
			dockerfilePath: "Dockerfile",
			expectedCode:   connect.CodeInternal,
			shouldContain: []string{
				"build failed",
			},
		},
		{
			name:           "network error",
			buildError:     errors.New("failed to fetch base image: network timeout"),
			dockerfilePath: "Dockerfile",
			expectedCode:   connect.CodeInternal,
			shouldContain: []string{
				"build failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyBuildError(tt.buildError, tt.dockerfilePath)

			// Check error is not nil
			if err == nil {
				t.Error("expected error, got nil")
				return
			}

			// Check it's a connect error
			connectErr, ok := err.(*connect.Error)
			if !ok {
				t.Errorf("expected *connect.Error, got %T", err)
				return
			}

			// Check error code
			if connectErr.Code() != tt.expectedCode {
				t.Errorf("expected code %v, got %v", tt.expectedCode, connectErr.Code())
			}

			// Check error message contains expected strings
			errorMsg := connectErr.Message()
			for _, contain := range tt.shouldContain {
				if !strings.Contains(errorMsg, contain) {
					t.Errorf("error message should contain %q, got: %s", contain, errorMsg)
				}
			}
		})
	}
}
