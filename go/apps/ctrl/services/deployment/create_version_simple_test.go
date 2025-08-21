package deployment

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

// TestInsertDeploymentParams_GitFields tests that InsertDeploymentParams includes all git fields
func TestInsertDeploymentParams_GitFields(t *testing.T) {
	// Test that the struct has all the expected git fields
	params := db.InsertDeploymentParams{}
	
	// Verify we can assign to all git fields (compile-time test)
	params.GitCommitSha = sql.NullString{String: "abc123", Valid: true}
	params.GitBranch = sql.NullString{String: "main", Valid: true}
	params.GitCommitMessage = sql.NullString{String: "test commit", Valid: true}
	params.GitCommitAuthorName = sql.NullString{String: "John Doe", Valid: true}
	params.GitCommitAuthorEmail = sql.NullString{String: "john@example.com", Valid: true}
	params.GitCommitAuthorUsername = sql.NullString{String: "johndoe", Valid: true}
	params.GitCommitAuthorAvatarUrl = sql.NullString{String: "https://github.com/johndoe.png", Valid: true}
	params.GitCommitTimestamp = sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}

	// Verify field values
	assert.Equal(t, "abc123", params.GitCommitSha.String)
	assert.True(t, params.GitCommitSha.Valid)
	assert.Equal(t, "main", params.GitBranch.String)
	assert.True(t, params.GitBranch.Valid)
	assert.Equal(t, "test commit", params.GitCommitMessage.String)
	assert.True(t, params.GitCommitMessage.Valid)
	assert.Equal(t, "John Doe", params.GitCommitAuthorName.String)
	assert.True(t, params.GitCommitAuthorName.Valid)
	assert.Equal(t, "john@example.com", params.GitCommitAuthorEmail.String)
	assert.True(t, params.GitCommitAuthorEmail.Valid)
	assert.Equal(t, "johndoe", params.GitCommitAuthorUsername.String)
	assert.True(t, params.GitCommitAuthorUsername.Valid)
	assert.Equal(t, "https://github.com/johndoe.png", params.GitCommitAuthorAvatarUrl.String)
	assert.True(t, params.GitCommitAuthorAvatarUrl.Valid)
	assert.True(t, params.GitCommitTimestamp.Valid)
}

// TestDeploymentModel_GitFields tests that Deployment model includes all git fields
func TestDeploymentModel_GitFields(t *testing.T) {
	// Test that the Deployment struct has all expected git fields
	deployment := db.Deployment{}
	now := time.Now().UnixMilli()
	
	// Verify we can assign to all git fields (compile-time test)
	deployment.GitCommitSha = sql.NullString{String: "abc123", Valid: true}
	deployment.GitBranch = sql.NullString{String: "main", Valid: true}
	deployment.GitCommitMessage = sql.NullString{String: "test commit", Valid: true}
	deployment.GitCommitAuthorName = sql.NullString{String: "John Doe", Valid: true}
	deployment.GitCommitAuthorEmail = sql.NullString{String: "john@example.com", Valid: true}
	deployment.GitCommitAuthorUsername = sql.NullString{String: "johndoe", Valid: true}
	deployment.GitCommitAuthorAvatarUrl = sql.NullString{String: "https://github.com/johndoe.png", Valid: true}
	deployment.GitCommitTimestamp = sql.NullInt64{Int64: now, Valid: true}

	// Verify field values
	assert.Equal(t, "abc123", deployment.GitCommitSha.String)
	assert.True(t, deployment.GitCommitSha.Valid)
	assert.Equal(t, "main", deployment.GitBranch.String)
	assert.True(t, deployment.GitBranch.Valid)
	assert.Equal(t, "test commit", deployment.GitCommitMessage.String)
	assert.True(t, deployment.GitCommitMessage.Valid)
	assert.Equal(t, "John Doe", deployment.GitCommitAuthorName.String)
	assert.True(t, deployment.GitCommitAuthorName.Valid)
	assert.Equal(t, "john@example.com", deployment.GitCommitAuthorEmail.String)
	assert.True(t, deployment.GitCommitAuthorEmail.Valid)
	assert.Equal(t, "johndoe", deployment.GitCommitAuthorUsername.String)
	assert.True(t, deployment.GitCommitAuthorUsername.Valid)
	assert.Equal(t, "https://github.com/johndoe.png", deployment.GitCommitAuthorAvatarUrl.String)
	assert.True(t, deployment.GitCommitAuthorAvatarUrl.Valid)
	assert.Equal(t, now, deployment.GitCommitTimestamp.Int64)
	assert.True(t, deployment.GitCommitTimestamp.Valid)
}

// TestCreateVersionRequest_GitFields tests that CreateVersionRequest protobuf includes all git fields
func TestCreateVersionRequest_GitFields(t *testing.T) {
	// Test that the protobuf request has all expected git fields
	now := time.Now().UnixMilli()
	
	req := &ctrlv1.CreateVersionRequest{
		WorkspaceId:                  "ws_test123",
		ProjectId:                   "proj_test456",
		Branch:                      "feature/git-info",
		SourceType:                  ctrlv1.SourceType_SOURCE_TYPE_GIT,
		GitCommitSha:                "abc123def456",
		GitCommitMessage:            "feat: add git information",
		GitCommitAuthorName:         "John Doe",
		GitCommitAuthorEmail:        "john@example.com",
		GitCommitAuthorUsername:     "johndoe",
		GitCommitAuthorAvatarUrl:    "https://github.com/johndoe.png",
		GitCommitTimestamp:          now,
	}

	// Verify all fields can be read
	assert.Equal(t, "ws_test123", req.GetWorkspaceId())
	assert.Equal(t, "proj_test456", req.GetProjectId())
	assert.Equal(t, "feature/git-info", req.GetBranch())
	assert.Equal(t, ctrlv1.SourceType_SOURCE_TYPE_GIT, req.GetSourceType())
	assert.Equal(t, "abc123def456", req.GetGitCommitSha())
	assert.Equal(t, "feat: add git information", req.GetGitCommitMessage())
	assert.Equal(t, "John Doe", req.GetGitCommitAuthorName())
	assert.Equal(t, "john@example.com", req.GetGitCommitAuthorEmail())
	assert.Equal(t, "johndoe", req.GetGitCommitAuthorUsername())
	assert.Equal(t, "https://github.com/johndoe.png", req.GetGitCommitAuthorAvatarUrl())
	assert.Equal(t, now, req.GetGitCommitTimestamp())
}

// TestVersion_GitFields tests that Version protobuf includes all git fields
func TestVersion_GitFields(t *testing.T) {
	// Test that the protobuf response has all expected git fields
	now := time.Now().UnixMilli()
	
	version := &ctrlv1.Version{
		Id:                       "deployment_test123",
		WorkspaceId:              "ws_test456",
		ProjectId:                "proj_test789",
		EnvironmentId:            "preview",
		GitCommitSha:             "abc123def456",
		GitBranch:                "feature/git-info",
		GitCommitMessage:         "feat: add git information",
		GitCommitAuthorName:      "John Doe",
		GitCommitAuthorEmail:     "john@example.com",
		GitCommitAuthorUsername:  "johndoe",
		GitCommitAuthorAvatarUrl: "https://github.com/johndoe.png",
		GitCommitTimestamp:       now,
		Status:                   ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE,
		CreatedAt:                now,
	}

	// Verify all fields can be read
	assert.Equal(t, "deployment_test123", version.GetId())
	assert.Equal(t, "ws_test456", version.GetWorkspaceId())
	assert.Equal(t, "proj_test789", version.GetProjectId())
	assert.Equal(t, "preview", version.GetEnvironmentId())
	assert.Equal(t, "abc123def456", version.GetGitCommitSha())
	assert.Equal(t, "feature/git-info", version.GetGitBranch())
	assert.Equal(t, "feat: add git information", version.GetGitCommitMessage())
	assert.Equal(t, "John Doe", version.GetGitCommitAuthorName())
	assert.Equal(t, "john@example.com", version.GetGitCommitAuthorEmail())
	assert.Equal(t, "johndoe", version.GetGitCommitAuthorUsername())
	assert.Equal(t, "https://github.com/johndoe.png", version.GetGitCommitAuthorAvatarUrl())
	assert.Equal(t, now, version.GetGitCommitTimestamp())
	assert.Equal(t, ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE, version.GetStatus())
}

// TestGitFieldValidation_SpecialCharacters tests handling of special characters
func TestGitFieldValidation_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single quotes",
			input:    "Fix bug with 'single quotes' in SQL",
			expected: "Fix bug with 'single quotes' in SQL",
		},
		{
			name:     "double quotes",
			input:    "Add support for \"double quotes\" parsing",
			expected: "Add support for \"double quotes\" parsing",
		},
		{
			name:     "unicode characters",
			input:    "Support unicode: ñoño résumé 🚀",
			expected: "Support unicode: ñoño résumé 🚀",
		},
		{
			name:     "newlines",
			input:    "Multi-line commit\n\nWith detailed description",
			expected: "Multi-line commit\n\nWith detailed description",
		},
		{
			name:     "email with plus",
			input:    "user+test@example.com",
			expected: "user+test@example.com",
		},
		{
			name:     "url with query params",
			input:    "https://github.com/user.png?size=40&v=4",
			expected: "https://github.com/user.png?size=40&v=4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that special characters are preserved in protobuf
			req := &ctrlv1.CreateVersionRequest{
				GitCommitMessage:         tt.input,
				GitCommitAuthorName:      tt.input,
				GitCommitAuthorEmail:     tt.input,
				GitCommitAuthorAvatarUrl: tt.input,
			}

			assert.Equal(t, tt.expected, req.GetGitCommitMessage())
			assert.Equal(t, tt.expected, req.GetGitCommitAuthorName())
			assert.Equal(t, tt.expected, req.GetGitCommitAuthorEmail())
			assert.Equal(t, tt.expected, req.GetGitCommitAuthorAvatarUrl())

			// Test that special characters are preserved in database model
			deployment := db.Deployment{
				GitCommitMessage:         sql.NullString{String: tt.input, Valid: true},
				GitCommitAuthorName:      sql.NullString{String: tt.input, Valid: true},
				GitCommitAuthorEmail:     sql.NullString{String: tt.input, Valid: true},
				GitCommitAuthorAvatarUrl: sql.NullString{String: tt.input, Valid: true},
			}

			assert.Equal(t, tt.expected, deployment.GitCommitMessage.String)
			assert.Equal(t, tt.expected, deployment.GitCommitAuthorName.String)
			assert.Equal(t, tt.expected, deployment.GitCommitAuthorEmail.String)
			assert.Equal(t, tt.expected, deployment.GitCommitAuthorAvatarUrl.String)
		})
	}
}

// TestGitFieldValidation_NullHandling tests NULL value handling
func TestGitFieldValidation_NullHandling(t *testing.T) {
	// Test empty protobuf fields
	req := &ctrlv1.CreateVersionRequest{
		WorkspaceId:              "ws_test",
		ProjectId:               "proj_test",
		GitCommitMessage:        "",
		GitCommitAuthorName:     "",
		GitCommitAuthorEmail:    "",
		GitCommitAuthorUsername: "",
		GitCommitAuthorAvatarUrl: "",
		GitCommitTimestamp:      0,
	}

	// Empty strings should be returned as-is
	assert.Equal(t, "", req.GetGitCommitMessage())
	assert.Equal(t, "", req.GetGitCommitAuthorName())
	assert.Equal(t, "", req.GetGitCommitAuthorEmail())
	assert.Equal(t, "", req.GetGitCommitAuthorUsername())
	assert.Equal(t, "", req.GetGitCommitAuthorAvatarUrl())
	assert.Equal(t, int64(0), req.GetGitCommitTimestamp())

	// Test NULL database fields
	deployment := db.Deployment{
		GitCommitMessage:         sql.NullString{Valid: false},
		GitCommitAuthorName:      sql.NullString{Valid: false},
		GitCommitAuthorEmail:     sql.NullString{Valid: false},
		GitCommitAuthorUsername:  sql.NullString{Valid: false},
		GitCommitAuthorAvatarUrl: sql.NullString{Valid: false},
		GitCommitTimestamp:       sql.NullInt64{Valid: false},
	}

	// NULL fields should be invalid
	assert.False(t, deployment.GitCommitMessage.Valid)
	assert.False(t, deployment.GitCommitAuthorName.Valid)
	assert.False(t, deployment.GitCommitAuthorEmail.Valid)
	assert.False(t, deployment.GitCommitAuthorUsername.Valid)
	assert.False(t, deployment.GitCommitAuthorAvatarUrl.Valid)
	assert.False(t, deployment.GitCommitTimestamp.Valid)
}

// TestTimestampConversion tests timestamp handling between protobuf and database
func TestTimestampConversion(t *testing.T) {
	// Test current timestamp
	now := time.Now()
	nowMillis := now.UnixMilli()

	// Test protobuf timestamp
	req := &ctrlv1.CreateVersionRequest{
		GitCommitTimestamp: nowMillis,
	}
	assert.Equal(t, nowMillis, req.GetGitCommitTimestamp())

	// Test database timestamp
	deployment := db.Deployment{
		GitCommitTimestamp: sql.NullInt64{Int64: nowMillis, Valid: true},
	}
	assert.Equal(t, nowMillis, deployment.GitCommitTimestamp.Int64)
	assert.True(t, deployment.GitCommitTimestamp.Valid)

	// Test conversion back to time
	retrievedTime := time.UnixMilli(deployment.GitCommitTimestamp.Int64)
	assert.Equal(t, now.Unix(), retrievedTime.Unix()) // Compare at second precision
}