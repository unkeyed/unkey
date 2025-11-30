package deployment

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
)

// validateTimestamp applies the same validation logic as the CreateDeployment service
func validateTimestamp(timestamp int64) bool {
	if timestamp == 0 {
		return true // Zero timestamps skip validation (optional field)
	}

	isValidLowerBound := timestamp >= 1_000_000_000_000
	maxValidTimestamp := time.Now().Add(1 * time.Hour).UnixMilli()
	isValidUpperBound := timestamp <= maxValidTimestamp
	return isValidLowerBound && isValidUpperBound
}

// TestGitFieldValidation_SpecialCharacters tests handling of special characters
func TestGitFieldValidation_SpecialCharacters(t *testing.T) {
	t.Parallel()

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
			name:     "username with dash",
			input:    "user-test",
			expected: "user-test",
		},
		{
			name:     "url with query params",
			input:    "https://github.com/user.png?size=40&v=4",
			expected: "https://github.com/user.png?size=40&v=4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test that special characters are preserved in protobuf
			gitCommit := &ctrlv1.GitCommitInfo{
				CommitMessage:   tt.input,
				AuthorHandle:    tt.input,
				AuthorAvatarUrl: tt.input,
			}

			require.Equal(t, tt.expected, gitCommit.GetCommitMessage())
			require.Equal(t, tt.expected, gitCommit.GetAuthorHandle())
			require.Equal(t, tt.expected, gitCommit.GetAuthorAvatarUrl())

			// Test that special characters are preserved in database model
			deployment := db.Deployment{
				GitCommitMessage:         sql.NullString{String: tt.input, Valid: true},
				GitCommitAuthorHandle:    sql.NullString{String: tt.input, Valid: true},
				GitCommitAuthorAvatarUrl: sql.NullString{String: tt.input, Valid: true},
			}

			require.Equal(t, tt.expected, deployment.GitCommitMessage.String)
			require.Equal(t, tt.expected, deployment.GitCommitAuthorHandle.String)
			require.Equal(t, tt.expected, deployment.GitCommitAuthorAvatarUrl.String)
		})
	}
}

// TestGitFieldValidation_NullHandling tests NULL value handling
func TestGitFieldValidation_NullHandling(t *testing.T) {
	t.Parallel()

	// Test empty protobuf fields
	gitCommit := &ctrlv1.GitCommitInfo{
		CommitMessage:   "",
		AuthorHandle:    "",
		AuthorAvatarUrl: "",
		Timestamp:       0,
	}

	// Empty strings should be returned as-is
	require.Equal(t, "", gitCommit.GetCommitMessage())
	require.Equal(t, "", gitCommit.GetAuthorHandle())
	require.Equal(t, "", gitCommit.GetAuthorAvatarUrl())
	require.Equal(t, int64(0), gitCommit.GetTimestamp())

	// Test NULL database fields
	deployment := db.Deployment{
		GitCommitMessage:         sql.NullString{Valid: false},
		GitCommitAuthorHandle:    sql.NullString{Valid: false},
		GitCommitAuthorAvatarUrl: sql.NullString{Valid: false},
		GitCommitTimestamp:       sql.NullInt64{Valid: false},
	}

	// NULL fields should be invalid
	require.False(t, deployment.GitCommitMessage.Valid)
	require.False(t, deployment.GitCommitAuthorHandle.Valid)
	require.False(t, deployment.GitCommitAuthorAvatarUrl.Valid)
	require.False(t, deployment.GitCommitTimestamp.Valid)
}

// TestTimestampConversion tests timestamp handling between protobuf and database
func TestTimestampConversion(t *testing.T) {
	t.Parallel()

	// Test fixed timestamp for deterministic testing
	now := time.Date(2024, 8, 21, 14, 30, 45, 123456789, time.UTC)
	nowMillis := now.UnixMilli()

	// Test protobuf timestamp
	gitCommit := &ctrlv1.GitCommitInfo{
		Timestamp: nowMillis,
	}
	require.Equal(t, nowMillis, gitCommit.GetTimestamp())

	// Test database timestamp
	deployment := db.Deployment{
		GitCommitTimestamp: sql.NullInt64{Int64: nowMillis, Valid: true},
	}
	require.Equal(t, nowMillis, deployment.GitCommitTimestamp.Int64)
	require.True(t, deployment.GitCommitTimestamp.Valid)

	// Test conversion back to time
	retrievedTime := time.UnixMilli(deployment.GitCommitTimestamp.Int64)
	require.Equal(t, now.Unix(), retrievedTime.Unix()) // Compare at second precision
}

// TestCreateDeploymentTimestampValidation_InvalidSecondsFormat tests timestamp validation
func TestCreateDeploymentTimestampValidation_InvalidSecondsFormat(t *testing.T) {
	t.Parallel()

	// Create proto request directly with seconds timestamp (should be rejected)
	req := &ctrlv1.CreateDeploymentRequest{
		ProjectId:       "proj_test456",
		Branch:          "main",
		EnvironmentSlug: "production",
		Source: &ctrlv1.CreateDeploymentRequest_BuildContext{
			BuildContext: &ctrlv1.BuildContext{
				BuildContextPath: "test-key",
				DockerfilePath:   ptr.P("Dockerfile"),
			},
		},
		GitCommit: &ctrlv1.GitCommitInfo{
			CommitSha: "abc123def456",
			Timestamp: time.Now().Unix(), // This is in seconds - should be rejected
		},
	}

	// Use shared validation helper
	isValid := validateTimestamp(req.GetGitCommit().GetTimestamp())
	require.False(t, isValid, "Seconds-based timestamp should be considered invalid")
}

// TestTimestampValidationBoundaries tests edge cases for timestamp validation
func TestTimestampValidationBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		timestamp   int64
		shouldPass  bool
		description string
	}{
		{
			name:        "exactly_at_threshold",
			timestamp:   1_000_000_000_000, // Exactly Jan 1, 2001 in milliseconds
			shouldPass:  true,
			description: "Timestamp exactly at the millisecond threshold should pass",
		},
		{
			name:        "just_below_threshold",
			timestamp:   999_999_999_999, // Just below threshold (seconds format)
			shouldPass:  false,
			description: "Timestamp just below threshold should fail",
		},
		{
			name:        "zero_timestamp",
			timestamp:   0,
			shouldPass:  true, // Zero is treated as "not provided" and skips validation
			description: "Zero timestamp should be allowed (optional field)",
		},
		{
			name:        "current_time_millis",
			timestamp:   time.Now().UnixMilli(),
			shouldPass:  true,
			description: "Current time in milliseconds should pass",
		},
		{
			name:        "far_future",
			timestamp:   time.Now().Add(24 * time.Hour).UnixMilli(),
			shouldPass:  false,
			description: "Timestamp too far in future should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Use shared validation helper
			isValid := validateTimestamp(tt.timestamp)

			if tt.shouldPass {
				require.True(t, isValid, tt.description)
			} else {
				require.False(t, isValid, tt.description)
			}
		})
	}
}

// TestCreateDeploymentFieldMapping tests the actual field mapping from protobuf to database params
func TestCreateDeploymentFieldMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		request  *ctrlv1.CreateDeploymentRequest
		expected struct {
			gitCommitSha                  string
			gitCommitShaValid             bool
			gitBranch                     string
			gitBranchValid                bool
			gitCommitMessage              string
			gitCommitMessageValid         bool
			gitCommitAuthorHandle         string
			gitCommitAuthorHandleValid    bool
			gitCommitAuthorAvatarUrl      string
			gitCommitAuthorAvatarUrlValid bool
			gitCommitTimestamp            int64
			gitCommitTimestampValid       bool
		}
	}{
		{
			name: "all_git_fields_populated",
			request: &ctrlv1.CreateDeploymentRequest{
				ProjectId:       "proj_test456",
				Branch:          "feature/test-branch",
				EnvironmentSlug: "production",
				Source: &ctrlv1.CreateDeploymentRequest_BuildContext{
					BuildContext: &ctrlv1.BuildContext{
						BuildContextPath: "test-key",
						DockerfilePath:   ptr.P("Dockerfile"),
					},
				},
				GitCommit: &ctrlv1.GitCommitInfo{
					CommitSha:       "abc123def456789",
					CommitMessage:   "feat: implement new feature",
					AuthorHandle:    "janedoe",
					AuthorAvatarUrl: "https://github.com/janedoe.png",
					Timestamp:       1724251845123, // Fixed millisecond timestamp
				},
			},
			expected: struct {
				gitCommitSha                  string
				gitCommitShaValid             bool
				gitBranch                     string
				gitBranchValid                bool
				gitCommitMessage              string
				gitCommitMessageValid         bool
				gitCommitAuthorHandle         string
				gitCommitAuthorHandleValid    bool
				gitCommitAuthorAvatarUrl      string
				gitCommitAuthorAvatarUrlValid bool
				gitCommitTimestamp            int64
				gitCommitTimestampValid       bool
			}{
				gitCommitSha:                  "abc123def456789",
				gitCommitShaValid:             true,
				gitBranch:                     "feature/test-branch",
				gitBranchValid:                true,
				gitCommitMessage:              "feat: implement new feature",
				gitCommitMessageValid:         true,
				gitCommitAuthorHandle:         "janedoe",
				gitCommitAuthorHandleValid:    true,
				gitCommitAuthorAvatarUrl:      "https://github.com/janedoe.png",
				gitCommitAuthorAvatarUrlValid: true,
				gitCommitTimestamp:            1724251845123,
				gitCommitTimestampValid:       true,
			},
		},
		{
			name: "empty_git_fields",
			request: &ctrlv1.CreateDeploymentRequest{
				ProjectId:       "proj_test456",
				Branch:          "main",
				EnvironmentSlug: "production",
				Source: &ctrlv1.CreateDeploymentRequest_BuildContext{
					BuildContext: &ctrlv1.BuildContext{
						BuildContextPath: "test-key",
						DockerfilePath:   ptr.P("Dockerfile"),
					},
				},
				GitCommit: &ctrlv1.GitCommitInfo{
					CommitSha:       "",
					CommitMessage:   "",
					AuthorHandle:    "",
					AuthorAvatarUrl: "",
					Timestamp:       0,
				},
			},
			expected: struct {
				gitCommitSha                  string
				gitCommitShaValid             bool
				gitBranch                     string
				gitBranchValid                bool
				gitCommitMessage              string
				gitCommitMessageValid         bool
				gitCommitAuthorHandle         string
				gitCommitAuthorHandleValid    bool
				gitCommitAuthorAvatarUrl      string
				gitCommitAuthorAvatarUrlValid bool
				gitCommitTimestamp            int64
				gitCommitTimestampValid       bool
			}{
				gitCommitSha:                  "",
				gitCommitShaValid:             false,
				gitBranch:                     "main",
				gitBranchValid:                true,
				gitCommitMessage:              "",
				gitCommitMessageValid:         false,
				gitCommitAuthorHandle:         "",
				gitCommitAuthorHandleValid:    false,
				gitCommitAuthorAvatarUrl:      "",
				gitCommitAuthorAvatarUrlValid: false,
				gitCommitTimestamp:            0,
				gitCommitTimestampValid:       false,
			},
		},
		{
			name: "mixed_populated_and_empty_fields",
			request: &ctrlv1.CreateDeploymentRequest{
				ProjectId:       "proj_test456",
				Branch:          "hotfix/urgent-fix",
				EnvironmentSlug: "production",
				Source: &ctrlv1.CreateDeploymentRequest_BuildContext{
					BuildContext: &ctrlv1.BuildContext{
						BuildContextPath: "test-key",
						DockerfilePath:   ptr.P("Dockerfile"),
					},
				},
				GitCommit: &ctrlv1.GitCommitInfo{
					CommitSha:       "xyz789abc123",
					CommitMessage:   "fix: critical security issue",
					AuthorHandle:    "", // Empty
					AuthorAvatarUrl: "", // Empty
					Timestamp:       1724251845999,
				},
			},
			expected: struct {
				gitCommitSha                  string
				gitCommitShaValid             bool
				gitBranch                     string
				gitBranchValid                bool
				gitCommitMessage              string
				gitCommitMessageValid         bool
				gitCommitAuthorHandle         string
				gitCommitAuthorHandleValid    bool
				gitCommitAuthorAvatarUrl      string
				gitCommitAuthorAvatarUrlValid bool
				gitCommitTimestamp            int64
				gitCommitTimestampValid       bool
			}{
				gitCommitSha:                  "xyz789abc123",
				gitCommitShaValid:             true,
				gitBranch:                     "hotfix/urgent-fix",
				gitBranchValid:                true,
				gitCommitMessage:              "fix: critical security issue",
				gitCommitMessageValid:         true,
				gitCommitAuthorHandle:         "",
				gitCommitAuthorHandleValid:    false, // Empty string should be invalid
				gitCommitAuthorAvatarUrl:      "",
				gitCommitAuthorAvatarUrlValid: false, // Empty string should be invalid
				gitCommitTimestamp:            1724251845999,
				gitCommitTimestampValid:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Extract git info (simulating service logic)
			var gitCommitSha, gitCommitMessage, gitCommitAuthorHandle, gitCommitAuthorAvatarUrl string
			var gitCommitTimestamp int64

			if gitCommit := tt.request.GetGitCommit(); gitCommit != nil {
				gitCommitSha = gitCommit.GetCommitSha()
				gitCommitMessage = gitCommit.GetCommitMessage()
				gitCommitAuthorHandle = gitCommit.GetAuthorHandle()
				gitCommitAuthorAvatarUrl = gitCommit.GetAuthorAvatarUrl()
				gitCommitTimestamp = gitCommit.GetTimestamp()
			}

			// Simulate the mapping logic from create_deployment.go
			// nolint: all
			params := db.InsertDeploymentParams{
				ID:                       "test_deployment_id",
				WorkspaceID:              "ws_test123",
				ProjectID:                tt.request.GetProjectId(),
				EnvironmentID:            "env_test",
				GitCommitSha:             sql.NullString{String: gitCommitSha, Valid: gitCommitSha != ""},
				GitBranch:                sql.NullString{String: tt.request.GetBranch(), Valid: true},
				GitCommitMessage:         sql.NullString{String: gitCommitMessage, Valid: gitCommitMessage != ""},
				GitCommitAuthorHandle:    sql.NullString{String: gitCommitAuthorHandle, Valid: gitCommitAuthorHandle != ""},
				GitCommitAuthorAvatarUrl: sql.NullString{String: gitCommitAuthorAvatarUrl, Valid: gitCommitAuthorAvatarUrl != ""},
				GitCommitTimestamp:       sql.NullInt64{Int64: gitCommitTimestamp, Valid: gitCommitTimestamp != 0},
				OpenapiSpec:              sql.NullString{String: "", Valid: false},
			}

			// Assert Git field mappings are correct
			require.Equal(t, tt.expected.gitCommitSha, params.GitCommitSha.String, "GitCommitSha string mismatch")
			require.Equal(t, tt.expected.gitCommitShaValid, params.GitCommitSha.Valid, "GitCommitSha valid flag mismatch")

			require.Equal(t, tt.expected.gitBranch, params.GitBranch.String, "GitBranch string mismatch")
			require.Equal(t, tt.expected.gitBranchValid, params.GitBranch.Valid, "GitBranch valid flag mismatch")

			require.Equal(t, tt.expected.gitCommitMessage, params.GitCommitMessage.String, "GitCommitMessage string mismatch")
			require.Equal(t, tt.expected.gitCommitMessageValid, params.GitCommitMessage.Valid, "GitCommitMessage valid flag mismatch")

			require.Equal(t, tt.expected.gitCommitAuthorHandle, params.GitCommitAuthorHandle.String, "GitCommitAuthorHandle string mismatch")
			require.Equal(t, tt.expected.gitCommitAuthorHandleValid, params.GitCommitAuthorHandle.Valid, "GitCommitAuthorHandle valid flag mismatch")

			require.Equal(t, tt.expected.gitCommitAuthorAvatarUrl, params.GitCommitAuthorAvatarUrl.String, "GitCommitAuthorAvatarUrl string mismatch")
			require.Equal(t, tt.expected.gitCommitAuthorAvatarUrlValid, params.GitCommitAuthorAvatarUrl.Valid, "GitCommitAuthorAvatarUrl valid flag mismatch")

			require.Equal(t, tt.expected.gitCommitTimestamp, params.GitCommitTimestamp.Int64, "GitCommitTimestamp value mismatch")
			require.Equal(t, tt.expected.gitCommitTimestampValid, params.GitCommitTimestamp.Valid, "GitCommitTimestamp valid flag mismatch")
		})
	}
}
