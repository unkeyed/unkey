package errors

import (
	"strings"
	"testing"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestParsePermissionError(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		expected []string
	}{
		{
			name:   "single permission",
			detail: "Missing one of these permissions: ['{ project.*.generate_upload_url []}']",
			expected: []string{
				"project.*.generate_upload_url",
			},
		},
		{
			name:   "multiple permissions",
			detail: "Missing one of these permissions: ['{ project.*.generate_upload_url []}', '{ project.proj_5VqYL7VzWnsgU8PA.generate_upload_url []}']",
			expected: []string{
				"project.*.generate_upload_url",
				"project.proj_5VqYL7VzWnsgU8PA.generate_upload_url",
			},
		},
		{
			name:   "permission with spaces",
			detail: "Missing one of these permissions: ['{ project.*.read_project [] }', '{ api.*.create_api [] }']",
			expected: []string{
				"project.*.read_project",
				"api.*.create_api",
			},
		},
		{
			name:     "no permissions found",
			detail:   "Some other error message",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePermissionError(tt.detail)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d permissions, got %d", len(tt.expected), len(result))
				return
			}

			for i, perm := range result {
				if perm != tt.expected[i] {
					t.Errorf("expected permission[%d] = %q, got %q", i, tt.expected[i], perm)
				}
			}
		})
	}
}

func TestFormatPermissionError(t *testing.T) {
	tests := []struct {
		name             string
		apiErr           *ParsedError
		expectedInMsg    []string // Strings that should appear in the message
		notExpectedInMsg []string // Strings that should NOT appear
	}{
		{
			name: "single permission error",
			apiErr: &ParsedError{
				Status: 403,
				Title:  "Insufficient Permissions",
				Detail: "Missing one of these permissions: ['{ project.*.generate_upload_url []}']",
				Type:   "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Missing required permissions",
				"You need this permission:",
				"project.*.generate_upload_url",
				"add one of these permissions to your root key",
			},
			notExpectedInMsg: []string{
				"have:",
				"[]",
				"{ ",
				" }",
			},
		},
		{
			name: "multiple permission error",
			apiErr: &ParsedError{
				Status: 403,
				Title:  "Insufficient Permissions",
				Detail: "Missing one of these permissions: ['{ project.*.generate_upload_url []}', '{ project.proj_123.generate_upload_url []}']",
				Type:   "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Missing required permissions",
				"You need ONE of the following permissions:",
				"project.*.generate_upload_url",
				"project.proj_123.generate_upload_url",
				"add one of these permissions to your root key",
			},
			notExpectedInMsg: []string{
				"have:",
				"[]",
				"{ ",
				" }",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPermissionError(tt.apiErr)

			// Check for expected strings
			for _, expected := range tt.expectedInMsg {
				if !strings.Contains(result, expected) {
					t.Errorf("expected message to contain %q, but it didn't.\nGot: %s", expected, result)
				}
			}

			// Check for strings that should NOT be present
			for _, notExpected := range tt.notExpectedInMsg {
				if strings.Contains(result, notExpected) {
					t.Errorf("expected message NOT to contain %q, but it did.\nGot: %s", notExpected, result)
				}
			}
		})
	}
}

func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		name     string
		apiErr   *ParsedError
		expected bool
	}{
		{
			name: "403 status code",
			apiErr: &ParsedError{
				Status: 403,
				Detail: "Some error",
			},
			expected: true,
		},
		{
			name: "not a permission error - 400",
			apiErr: &ParsedError{
				Status: 400,
				Title:  "Bad Request",
				Detail: "Invalid input",
			},
			expected: false,
		},
		{
			name: "not a permission error - 401",
			apiErr: &ParsedError{
				Status: 401,
				Title:  "Unauthorized",
				Detail: "Invalid token",
			},
			expected: false,
		},
		{
			name: "not a permission error - 404",
			apiErr: &ParsedError{
				Status: 404,
				Title:  "Not Found",
				Detail: "Resource not found",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPermissionError(tt.apiErr)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatAuthenticationError(t *testing.T) {
	tests := []struct {
		name             string
		apiErr           *ParsedError
		expectedInMsg    []string
		notExpectedInMsg []string
	}{
		{
			name: "invalid token",
			apiErr: &ParsedError{
				Status: 401,
				Title:  "Unauthorized",
				Detail: "Invalid token provided",
				Type:   "https://unkey.com/docs/errors/unkey/authorization/invalid_token",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Authentication failed",
				"root key is invalid or has expired",
				"--root-key flag or UNKEY_ROOT_KEY env var",
			},
		},
		{
			name: "generic auth error",
			apiErr: &ParsedError{
				Status: 401,
				Title:  "Unauthorized",
				Detail: "Authentication required",
				Type:   "https://unkey.com/docs/errors/unkey/authorization/unauthorized",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Authentication failed",
				"Could not authenticate your request",
				"--root-key flag",
				"UNKEY_ROOT_KEY environment variable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAuthenticationError(tt.apiErr)

			for _, expected := range tt.expectedInMsg {
				if !strings.Contains(result, expected) {
					t.Errorf("expected message to contain %q, but it didn't.\nGot: %s", expected, result)
				}
			}

			if tt.notExpectedInMsg != nil {
				for _, notExpected := range tt.notExpectedInMsg {
					if strings.Contains(result, notExpected) {
						t.Errorf("expected message NOT to contain %q, but it did.\nGot: %s", notExpected, result)
					}
				}
			}
		})
	}
}

func TestFormatNotFoundError(t *testing.T) {
	tests := []struct {
		name          string
		apiErr        *ParsedError
		expectedInMsg []string
	}{
		{
			name: "deployment not found",
			apiErr: &ParsedError{
				Status: 404,
				Title:  "Not Found",
				Detail: "The requested deployment does not exist or has been deleted.",
				Type:   "https://unkey.com/docs/errors/unkey/data/deployment_not_found",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Deployment not found",
				"doesn't exist or you don't have access to it",
			},
		},
		{
			name: "project not found",
			apiErr: &ParsedError{
				Status: 404,
				Title:  "Not Found",
				Detail: "The requested project does not exist or has been deleted.",
				Type:   "https://unkey.com/docs/errors/unkey/data/project_not_found",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Project not found",
				"doesn't exist or you don't have access to it",
			},
		},
		{
			name: "environment not found",
			apiErr: &ParsedError{
				Status: 404,
				Title:  "Not Found",
				Detail: "Environment not found.",
				Type:   "https://unkey.com/docs/errors/unkey/data/environment_not_found",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Environment not found",
				"doesn't exist in this project",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNotFoundError(tt.apiErr)

			for _, expected := range tt.expectedInMsg {
				if !strings.Contains(result, expected) {
					t.Errorf("expected message to contain %q, but it didn't.\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	tests := []struct {
		name          string
		apiErr        *ParsedError
		expectedInMsg []string
	}{
		{
			name: "validation error without fix",
			apiErr: &ParsedError{
				Status: 400,
				Title:  "Bad Request",
				Detail: "Validation failed",
				Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
				Meta:   components.Meta{},
				ValidationErrors: []components.ValidationError{
					{
						Location: "projectId",
						Message:  "Invalid project ID format",
					},
				},
			},
			expectedInMsg: []string{
				"Validation failed",
				"Validation errors:",
				"projectId",
				"Invalid project ID format",
			},
		},
		{
			name: "multiple validation errors",
			apiErr: &ParsedError{
				Status: 400,
				Title:  "Bad Request",
				Detail: "Multiple validation errors occurred",
				Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
				Meta:   components.Meta{},
				ValidationErrors: []components.ValidationError{
					{
						Location: "branch",
						Message:  "Field is required",
					},
					{
						Location: "projectId",
						Message:  "Invalid format",
					},
				},
			},
			expectedInMsg: []string{
				"Multiple validation errors occurred",
				"Validation errors:",
				"branch",
				"Field is required",
				"projectId",
				"Invalid format",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValidationError(tt.apiErr)

			for _, expected := range tt.expectedInMsg {
				if !strings.Contains(result, expected) {
					t.Errorf("expected message to contain %q, but it didn't.\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestFormatAPIError(t *testing.T) {
	tests := []struct {
		name          string
		apiErr        *ParsedError
		expectedInMsg []string
	}{
		{
			name: "permission error routes to permission formatter",
			apiErr: &ParsedError{
				Status: 403,
				Detail: "Missing one of these permissions: ['{ project.*.create_deployment []}']",
				Type:   "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Missing required permissions",
				"project.*.create_deployment",
			},
		},
		{
			name: "auth error routes to auth formatter",
			apiErr: &ParsedError{
				Status: 401,
				Detail: "Invalid token",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Authentication failed",
			},
		},
		{
			name: "not found error routes to not found formatter",
			apiErr: &ParsedError{
				Status: 404,
				Detail: "The requested project does not exist",
				Type:   "https://unkey.com/docs/errors/unkey/data/project_not_found",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Project not found",
			},
		},
		{
			name: "validation error routes to validation formatter",
			apiErr: &ParsedError{
				Status: 400,
				Detail: "Validation failed",
				Meta:   components.Meta{},
				ValidationErrors: []components.ValidationError{
					{
						Location: "branch",
						Message:  "Field is required",
					},
				},
			},
			expectedInMsg: []string{
				"Validation failed",
				"Validation errors:",
			},
		},
		{
			name: "generic error uses default formatter",
			apiErr: &ParsedError{
				Status: 500,
				Detail: "Internal server error",
				Meta:   components.Meta{},
			},
			expectedInMsg: []string{
				"Internal server error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAPIError(tt.apiErr)

			for _, expected := range tt.expectedInMsg {
				if !strings.Contains(result, expected) {
					t.Errorf("expected message to contain %q, but it didn't.\nGot: %s", expected, result)
				}
			}
		})
	}
}
