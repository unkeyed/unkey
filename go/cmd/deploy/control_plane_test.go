package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateDockerfile(t *testing.T) {
	// Create temporary test directories
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		setup          func() (contextPath, dockerfilePath string)
		expectedError  string
		shouldContain  []string
		shouldNotError bool
	}{
		{
			name: "valid dockerfile exists",
			setup: func() (string, string) {
				dir := filepath.Join(tempDir, "valid")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				dockerfilePath := filepath.Join(dir, "Dockerfile")
				if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir, "Dockerfile"
			},
			shouldNotError: true,
		},
		{
			name: "missing dockerfile with default name",
			setup: func() (string, string) {
				dir := filepath.Join(tempDir, "missing")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				return dir, "Dockerfile"
			},
			shouldContain: []string{
				"dockerfile not found",
				"Please ensure:",
				"Dockerfile exists in your build context directory",
				"--dockerfile flag",
				"--context flag",
			},
		},
		{
			name: "missing custom dockerfile",
			setup: func() (string, string) {
				dir := filepath.Join(tempDir, "missing-custom")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				return dir, "Dockerfile.prod"
			},
			shouldContain: []string{
				"dockerfile not found",
				"Dockerfile.prod",
			},
		},
		{
			name: "dockerfile path is a directory",
			setup: func() (string, string) {
				dir := filepath.Join(tempDir, "dir-instead")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				dockerfileDir := filepath.Join(dir, "Dockerfile")
				if err := os.MkdirAll(dockerfileDir, 0755); err != nil {
					t.Fatal(err)
				}
				return dir, "Dockerfile"
			},
			shouldContain: []string{
				"is a directory, not a file",
			},
		},
		{
			name: "valid custom dockerfile",
			setup: func() (string, string) {
				dir := filepath.Join(tempDir, "custom")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				dockerfilePath := filepath.Join(dir, "Dockerfile.custom")
				if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir, "Dockerfile.custom"
			},
			shouldNotError: true,
		},
		{
			name: "dockerfile in subdirectory",
			setup: func() (string, string) {
				dir := filepath.Join(tempDir, "subdir")
				if err := os.MkdirAll(filepath.Join(dir, "docker"), 0755); err != nil {
					t.Fatal(err)
				}
				dockerfilePath := filepath.Join(dir, "docker", "Dockerfile")
				if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir, "docker/Dockerfile"
			},
			shouldNotError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextPath, dockerfilePath := tt.setup()
			err := validateDockerfile(contextPath, dockerfilePath)

			if tt.shouldNotError {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Error("expected error, got nil")
				return
			}

			errorMsg := err.Error()
			for _, contain := range tt.shouldContain {
				if !strings.Contains(errorMsg, contain) {
					t.Errorf("error message should contain %q, got: %s", contain, errorMsg)
				}
			}
		})
	}
}

func TestValidateDockerfileWithRealPaths(t *testing.T) {
	// Test with the actual test directories we created
	tests := []struct {
		name           string
		contextPath    string
		dockerfilePath string
		shouldError    bool
		errorContains  string
	}{
		{
			name:           "missing dockerfile directory",
			contextPath:    "/tmp/test-dockerfile-validation/missing-dockerfile",
			dockerfilePath: "Dockerfile",
			shouldError:    true,
			errorContains:  "dockerfile not found",
		},
		{
			name:           "existing dockerfile directory",
			contextPath:    "/tmp/test-dockerfile-validation/with-dockerfile",
			dockerfilePath: "Dockerfile",
			shouldError:    false,
		},
		{
			name:           "custom dockerfile wrong path",
			contextPath:    "/tmp/test-dockerfile-validation/custom-dockerfile",
			dockerfilePath: "Dockerfile",
			shouldError:    true,
			errorContains:  "dockerfile not found",
		},
		{
			name:           "custom dockerfile correct path",
			contextPath:    "/tmp/test-dockerfile-validation/custom-dockerfile",
			dockerfilePath: "Dockerfile.custom",
			shouldError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if the test directory doesn't exist
			if _, err := os.Stat(tt.contextPath); os.IsNotExist(err) {
				t.Skipf("test directory %s does not exist", tt.contextPath)
				return
			}

			err := validateDockerfile(tt.contextPath, tt.dockerfilePath)

			if tt.shouldError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error should contain %q, got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}
