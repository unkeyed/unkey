package errors

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestFormatPermissionError(t *testing.T) {
	tests := []struct {
		name          string
		apiErr        *apierrors.ForbiddenErrorResponse
		expectedInMsg []string
	}{
		{
			name: "permission error",
			apiErr: &apierrors.ForbiddenErrorResponse{
				Meta: components.Meta{},
				Error_: components.BaseError{
					Detail: "Missing one of these permissions: ['{ project.*.generate_upload_url []}']",
					Title:  "Insufficient Permissions",
					Type:   "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions",
					Status: 403,
				},
			},
			expectedInMsg: []string{
				"Permission denied",
				"Missing one of these permissions",
				"update your root key permissions",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPermissionError(tt.apiErr)

			for _, expected := range tt.expectedInMsg {
				require.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatAuthenticationError(t *testing.T) {
	tests := []struct {
		name          string
		apiErr        *apierrors.UnauthorizedErrorResponse
		expectedInMsg []string
	}{
		{
			name: "invalid token",
			apiErr: &apierrors.UnauthorizedErrorResponse{
				Meta: components.Meta{},
				Error_: components.BaseError{
					Detail: "Invalid token provided",
					Title:  "Unauthorized",
					Type:   "https://unkey.com/docs/errors/unkey/authorization/invalid_token",
					Status: 401,
				},
			},
			expectedInMsg: []string{
				"Authentication failed",
				"root key is invalid or has expired",
				"--root-key flag or UNKEY_ROOT_KEY env var",
			},
		},
		{
			name: "generic auth error",
			apiErr: &apierrors.UnauthorizedErrorResponse{
				Meta: components.Meta{},
				Error_: components.BaseError{
					Detail: "Authentication required",
					Title:  "Unauthorized",
					Type:   "https://unkey.com/docs/errors/unkey/authorization/unauthorized",
					Status: 401,
				},
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
				require.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatNotFoundError(t *testing.T) {
	tests := []struct {
		name          string
		apiErr        *apierrors.NotFoundErrorResponse
		expectedInMsg []string
	}{
		{
			name: "deployment not found",
			apiErr: &apierrors.NotFoundErrorResponse{
				Meta: components.Meta{},
				Error_: components.BaseError{
					Detail: "The requested deployment does not exist or has been deleted.",
					Title:  "Not Found",
					Type:   "https://unkey.com/docs/errors/unkey/data/deployment_not_found",
					Status: 404,
				},
			},
			expectedInMsg: []string{
				"Deployment not found",
				"doesn't exist or you don't have access to it",
			},
		},
		{
			name: "project not found",
			apiErr: &apierrors.NotFoundErrorResponse{
				Meta: components.Meta{},
				Error_: components.BaseError{
					Detail: "The requested project does not exist or has been deleted.",
					Title:  "Not Found",
					Type:   "https://unkey.com/docs/errors/unkey/data/project_not_found",
					Status: 404,
				},
			},
			expectedInMsg: []string{
				"Project not found",
				"doesn't exist or you don't have access to it",
			},
		},
		{
			name: "environment not found",
			apiErr: &apierrors.NotFoundErrorResponse{
				Meta: components.Meta{},
				Error_: components.BaseError{
					Detail: "Environment not found.",
					Title:  "Not Found",
					Type:   "https://unkey.com/docs/errors/unkey/data/environment_not_found",
					Status: 404,
				},
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
				require.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	tests := []struct {
		name          string
		apiErr        *apierrors.BadRequestErrorResponse
		expectedInMsg []string
	}{
		{
			name: "validation error without fix",
			apiErr: &apierrors.BadRequestErrorResponse{
				Meta: components.Meta{},
				Error_: components.BadRequestErrorDetails{
					Detail: "Validation failed",
					Title:  "Bad Request",
					Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
					Status: 400,
					Errors: []components.ValidationError{
						{
							Location: "projectId",
							Message:  "Invalid project ID format",
						},
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
			apiErr: &apierrors.BadRequestErrorResponse{
				Meta: components.Meta{},
				Error_: components.BadRequestErrorDetails{
					Detail: "Multiple validation errors occurred",
					Title:  "Bad Request",
					Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
					Status: 400,
					Errors: []components.ValidationError{
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
				require.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedInMsg []string
	}{
		{
			name: "permission error routes to permission formatter",
			err: &apierrors.ForbiddenErrorResponse{
				Meta: components.Meta{},
				Error_: components.BaseError{
					Detail: "Missing one of these permissions: ['{ project.*.create_deployment []}']",
					Type:   "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions",
					Status: 403,
				},
			},
			expectedInMsg: []string{
				"Permission denied",
				"Missing one of these permissions",
			},
		},
		{
			name: "auth error routes to auth formatter",
			err: &apierrors.UnauthorizedErrorResponse{
				Meta: components.Meta{},
				Error_: components.BaseError{
					Detail: "Invalid token",
					Status: 401,
				},
			},
			expectedInMsg: []string{
				"Authentication failed",
			},
		},
		{
			name: "not found error routes to not found formatter",
			err: &apierrors.NotFoundErrorResponse{
				Meta: components.Meta{},
				Error_: components.BaseError{
					Detail: "The requested project does not exist",
					Type:   "https://unkey.com/docs/errors/unkey/data/project_not_found",
					Status: 404,
				},
			},
			expectedInMsg: []string{
				"Project not found",
			},
		},
		{
			name: "validation error routes to validation formatter",
			err: &apierrors.BadRequestErrorResponse{
				Meta: components.Meta{},
				Error_: components.BadRequestErrorDetails{
					Detail: "Validation failed",
					Status: 400,
					Errors: []components.ValidationError{
						{
							Location: "branch",
							Message:  "Field is required",
						},
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
			err: &apierrors.InternalServerErrorResponse{
				Meta: components.Meta{},
				Error_: components.BaseError{
					Detail: "Internal server error",
					Status: 500,
				},
			},
			expectedInMsg: []string{
				"Internal server error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatError(tt.err)

			for _, expected := range tt.expectedInMsg {
				require.Contains(t, result, expected)
			}
		})
	}
}
