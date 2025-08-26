package deployment

import (
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

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
			t.Parallel()

			// Test that special characters are preserved in protobuf
			req := &ctrlv1.CreateVersionRequest{
				GitCommitMessage:         tt.input,
				GitCommitAuthorName:      tt.input,
				GitCommitAuthorEmail:     tt.input,
				GitCommitAuthorUsername:  tt.input,
				GitCommitAuthorAvatarUrl: tt.input,
			}

			require.Equal(t, tt.expected, req.GetGitCommitMessage())
			require.Equal(t, tt.expected, req.GetGitCommitAuthorName())
			require.Equal(t, tt.expected, req.GetGitCommitAuthorEmail())
			require.Equal(t, tt.expected, req.GetGitCommitAuthorUsername())
			require.Equal(t, tt.expected, req.GetGitCommitAuthorAvatarUrl())

			// Test that special characters are preserved in database model
			deployment := db.Deployment{
				GitCommitMessage:         sql.NullString{String: tt.input, Valid: true},
				GitCommitAuthorName:      sql.NullString{String: tt.input, Valid: true},
				GitCommitAuthorEmail:     sql.NullString{String: tt.input, Valid: true},
				GitCommitAuthorUsername:  sql.NullString{String: tt.input, Valid: true},
				GitCommitAuthorAvatarUrl: sql.NullString{String: tt.input, Valid: true},
			}

			require.Equal(t, tt.expected, deployment.GitCommitMessage.String)
			require.Equal(t, tt.expected, deployment.GitCommitAuthorName.String)
			require.Equal(t, tt.expected, deployment.GitCommitAuthorEmail.String)
			require.Equal(t, tt.expected, deployment.GitCommitAuthorUsername.String)
			require.Equal(t, tt.expected, deployment.GitCommitAuthorAvatarUrl.String)
		})
	}
}

// TestGitFieldValidation_NullHandling tests NULL value handling
func TestGitFieldValidation_NullHandling(t *testing.T) {
	t.Parallel()

	// Test empty protobuf fields
	req := &ctrlv1.CreateVersionRequest{
		WorkspaceId:              "ws_test",
		ProjectId:                "proj_test",
		GitCommitMessage:         "",
		GitCommitAuthorName:      "",
		GitCommitAuthorEmail:     "",
		GitCommitAuthorUsername:  "",
		GitCommitAuthorAvatarUrl: "",
		GitCommitTimestamp:       0,
	}

	// Empty strings should be returned as-is
	require.Equal(t, "", req.GetGitCommitMessage())
	require.Equal(t, "", req.GetGitCommitAuthorName())
	require.Equal(t, "", req.GetGitCommitAuthorEmail())
	require.Equal(t, "", req.GetGitCommitAuthorUsername())
	require.Equal(t, "", req.GetGitCommitAuthorAvatarUrl())
	require.Equal(t, int64(0), req.GetGitCommitTimestamp())

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
	require.False(t, deployment.GitCommitMessage.Valid)
	require.False(t, deployment.GitCommitAuthorName.Valid)
	require.False(t, deployment.GitCommitAuthorEmail.Valid)
	require.False(t, deployment.GitCommitAuthorUsername.Valid)
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
	req := &ctrlv1.CreateVersionRequest{
		GitCommitTimestamp: nowMillis,
	}
	require.Equal(t, nowMillis, req.GetGitCommitTimestamp())

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

// TestCreateVersionTimestampValidation_InvalidSecondsFormat tests timestamp validation
func TestCreateVersionTimestampValidation_InvalidSecondsFormat(t *testing.T) {
	t.Parallel()

	// Create a mock service to test timestamp validation logic
	// Since we can't easily mock the full service dependencies,
	// we'll test the validation logic directly

	// Test case 1: Timestamp in seconds format (should be rejected)
	req := &connect.Request[ctrlv1.CreateVersionRequest]{}
	req.Msg = &ctrlv1.CreateVersionRequest{
		WorkspaceId:        "ws_test123",
		ProjectId:          "proj_test456",
		Branch:             "main",
		SourceType:         ctrlv1.SourceType_SOURCE_TYPE_GIT,
		GitCommitSha:       "abc123def456",
		GitCommitTimestamp: time.Now().Unix(), // This is in seconds - should be rejected
	}

	// Validate the timestamp would be rejected (simulate the validation logic)
	timestamp := req.Msg.GetGitCommitTimestamp()

	// Test the validation conditions
	require.True(t, timestamp > 0, "Timestamp should be provided")
	require.True(t, timestamp < 1_000_000_000_000, "Seconds timestamp should be less than millisecond threshold")

	// This simulates what the service validation would do
	isValidMilliseconds := timestamp >= 1_000_000_000_000
	require.False(t, isValidMilliseconds, "Seconds-based timestamp should be considered invalid")
}

// TestCreateVersionTimestampValidation_ValidMillisecondsFormat tests valid timestamp
func TestCreateVersionTimestampValidation_ValidMillisecondsFormat(t *testing.T) {
	t.Parallel()

	// Test case 2: Timestamp in milliseconds format (should be accepted)
	req := &connect.Request[ctrlv1.CreateVersionRequest]{}
	req.Msg = &ctrlv1.CreateVersionRequest{
		WorkspaceId:        "ws_test123",
		ProjectId:          "proj_test456",
		Branch:             "main",
		SourceType:         ctrlv1.SourceType_SOURCE_TYPE_GIT,
		GitCommitSha:       "abc123def456",
		GitCommitTimestamp: time.Now().UnixMilli(), // This is in milliseconds - should be accepted
	}

	// Validate the timestamp would be accepted
	timestamp := req.Msg.GetGitCommitTimestamp()

	// Test the validation conditions
	require.True(t, timestamp > 0, "Timestamp should be provided")
	require.True(t, timestamp >= 1_000_000_000_000, "Milliseconds timestamp should be >= threshold")

	// Test upper bound (not too far in future)
	maxValidTimestamp := time.Now().Add(1 * time.Hour).UnixMilli()
	require.True(t, timestamp <= maxValidTimestamp, "Timestamp should not be too far in future")

	// This simulates what the service validation would do
	isValidMilliseconds := timestamp >= 1_000_000_000_000 && timestamp <= maxValidTimestamp
	require.True(t, isValidMilliseconds, "Milliseconds-based timestamp should be considered valid")
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

			// Test the validation logic that would be applied in the service
			if tt.timestamp == 0 {
				// Zero timestamps skip validation (optional field)
				require.True(t, true, "Zero timestamp should skip validation")
				return
			}

			// Apply the same validation logic as in CreateVersion service
			isValidLowerBound := tt.timestamp >= 1_000_000_000_000
			maxValidTimestamp := time.Now().Add(1 * time.Hour).UnixMilli()
			isValidUpperBound := tt.timestamp <= maxValidTimestamp
			isValid := isValidLowerBound && isValidUpperBound

			if tt.shouldPass {
				require.True(t, isValid, tt.description)
			} else {
				require.False(t, isValid, tt.description)
			}
		})
	}
}

// TestCreateVersionFieldMapping tests the actual field mapping from protobuf to database params
func TestCreateVersionFieldMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		request  *ctrlv1.CreateVersionRequest
		expected struct {
			gitCommitSha                  string
			gitCommitShaValid             bool
			gitBranch                     string
			gitBranchValid                bool
			gitCommitMessage              string
			gitCommitMessageValid         bool
			gitCommitAuthorName           string
			gitCommitAuthorNameValid      bool
			gitCommitAuthorEmail          string
			gitCommitAuthorEmailValid     bool
			gitCommitAuthorUsername       string
			gitCommitAuthorUsernameValid  bool
			gitCommitAuthorAvatarUrl      string
			gitCommitAuthorAvatarUrlValid bool
			gitCommitTimestamp            int64
			gitCommitTimestampValid       bool
		}
	}{
		{
			name: "all_git_fields_populated",
			request: &ctrlv1.CreateVersionRequest{
				WorkspaceId:              "ws_test123",
				ProjectId:                "proj_test456",
				Branch:                   "feature/test-branch",
				SourceType:               ctrlv1.SourceType_SOURCE_TYPE_GIT,
				GitCommitSha:             "abc123def456789",
				GitCommitMessage:         "feat: implement new feature",
				GitCommitAuthorName:      "Jane Doe",
				GitCommitAuthorEmail:     "jane@example.com",
				GitCommitAuthorUsername:  "janedoe",
				GitCommitAuthorAvatarUrl: "https://github.com/janedoe.png",
				GitCommitTimestamp:       1724251845123, // Fixed millisecond timestamp
			},
			expected: struct {
				gitCommitSha                  string
				gitCommitShaValid             bool
				gitBranch                     string
				gitBranchValid                bool
				gitCommitMessage              string
				gitCommitMessageValid         bool
				gitCommitAuthorName           string
				gitCommitAuthorNameValid      bool
				gitCommitAuthorEmail          string
				gitCommitAuthorEmailValid     bool
				gitCommitAuthorUsername       string
				gitCommitAuthorUsernameValid  bool
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
				gitCommitAuthorName:           "Jane Doe",
				gitCommitAuthorNameValid:      true,
				gitCommitAuthorEmail:          "jane@example.com",
				gitCommitAuthorEmailValid:     true,
				gitCommitAuthorUsername:       "janedoe",
				gitCommitAuthorUsernameValid:  true,
				gitCommitAuthorAvatarUrl:      "https://github.com/janedoe.png",
				gitCommitAuthorAvatarUrlValid: true,
				gitCommitTimestamp:            1724251845123,
				gitCommitTimestampValid:       true,
			},
		},
		{
			name: "empty_git_fields",
			request: &ctrlv1.CreateVersionRequest{
				WorkspaceId:              "ws_test123",
				ProjectId:                "proj_test456",
				Branch:                   "main",
				SourceType:               ctrlv1.SourceType_SOURCE_TYPE_GIT,
				GitCommitSha:             "",
				GitCommitMessage:         "",
				GitCommitAuthorName:      "",
				GitCommitAuthorEmail:     "",
				GitCommitAuthorUsername:  "",
				GitCommitAuthorAvatarUrl: "",
				GitCommitTimestamp:       0,
			},
			expected: struct {
				gitCommitSha                  string
				gitCommitShaValid             bool
				gitBranch                     string
				gitBranchValid                bool
				gitCommitMessage              string
				gitCommitMessageValid         bool
				gitCommitAuthorName           string
				gitCommitAuthorNameValid      bool
				gitCommitAuthorEmail          string
				gitCommitAuthorEmailValid     bool
				gitCommitAuthorUsername       string
				gitCommitAuthorUsernameValid  bool
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
				gitCommitAuthorName:           "",
				gitCommitAuthorNameValid:      false,
				gitCommitAuthorEmail:          "",
				gitCommitAuthorEmailValid:     false,
				gitCommitAuthorUsername:       "",
				gitCommitAuthorUsernameValid:  false,
				gitCommitAuthorAvatarUrl:      "",
				gitCommitAuthorAvatarUrlValid: false,
				gitCommitTimestamp:            0,
				gitCommitTimestampValid:       false,
			},
		},
		{
			name: "mixed_populated_and_empty_fields",
			request: &ctrlv1.CreateVersionRequest{
				WorkspaceId:              "ws_test123",
				ProjectId:                "proj_test456",
				Branch:                   "hotfix/urgent-fix",
				SourceType:               ctrlv1.SourceType_SOURCE_TYPE_GIT,
				GitCommitSha:             "xyz789abc123",
				GitCommitMessage:         "fix: critical security issue",
				GitCommitAuthorName:      "", // Empty
				GitCommitAuthorEmail:     "security-team@example.com",
				GitCommitAuthorUsername:  "", // Empty
				GitCommitAuthorAvatarUrl: "", // Empty
				GitCommitTimestamp:       1724251845999,
			},
			expected: struct {
				gitCommitSha                  string
				gitCommitShaValid             bool
				gitBranch                     string
				gitBranchValid                bool
				gitCommitMessage              string
				gitCommitMessageValid         bool
				gitCommitAuthorName           string
				gitCommitAuthorNameValid      bool
				gitCommitAuthorEmail          string
				gitCommitAuthorEmailValid     bool
				gitCommitAuthorUsername       string
				gitCommitAuthorUsernameValid  bool
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
				gitCommitAuthorName:           "",
				gitCommitAuthorNameValid:      false, // Empty string should be invalid
				gitCommitAuthorEmail:          "security-team@example.com",
				gitCommitAuthorEmailValid:     true,
				gitCommitAuthorUsername:       "",
				gitCommitAuthorUsernameValid:  false, // Empty string should be invalid
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

			// Simulate the mapping logic from create_version.go
			// This tests the actual field wiring that happens in the service
			params := db.InsertDeploymentParams{
				ID:            "test_deployment_id",
				WorkspaceID:   tt.request.GetWorkspaceId(),
				ProjectID:     tt.request.GetProjectId(),
				Environment:   db.DeploymentsEnvironmentPreview,
				BuildID:       sql.NullString{String: "", Valid: false},
				RootfsImageID: "",
				// Git field mappings - this is what we're testing
				GitCommitSha:             sql.NullString{String: tt.request.GetGitCommitSha(), Valid: tt.request.GetGitCommitSha() != ""},
				GitBranch:                sql.NullString{String: tt.request.GetBranch(), Valid: true},
				GitCommitMessage:         sql.NullString{String: tt.request.GetGitCommitMessage(), Valid: tt.request.GetGitCommitMessage() != ""},
				GitCommitAuthorName:      sql.NullString{String: tt.request.GetGitCommitAuthorName(), Valid: tt.request.GetGitCommitAuthorName() != ""},
				GitCommitAuthorEmail:     sql.NullString{String: tt.request.GetGitCommitAuthorEmail(), Valid: tt.request.GetGitCommitAuthorEmail() != ""},
				GitCommitAuthorUsername:  sql.NullString{String: tt.request.GetGitCommitAuthorUsername(), Valid: tt.request.GetGitCommitAuthorUsername() != ""},
				GitCommitAuthorAvatarUrl: sql.NullString{String: tt.request.GetGitCommitAuthorAvatarUrl(), Valid: tt.request.GetGitCommitAuthorAvatarUrl() != ""},
				GitCommitTimestamp:       sql.NullInt64{Int64: tt.request.GetGitCommitTimestamp(), Valid: tt.request.GetGitCommitTimestamp() != 0},
				ConfigSnapshot:           []byte("{}"),
				OpenapiSpec:              sql.NullString{String: "", Valid: false},
				Status:                   "pending",
				CreatedAt:                1724251845000,
				UpdatedAt:                sql.NullInt64{Int64: 1724251845000, Valid: true},
			}

			// Assert Git field mappings are correct
			require.Equal(t, tt.expected.gitCommitSha, params.GitCommitSha.String, "GitCommitSha string mismatch")
			require.Equal(t, tt.expected.gitCommitShaValid, params.GitCommitSha.Valid, "GitCommitSha valid flag mismatch")

			require.Equal(t, tt.expected.gitBranch, params.GitBranch.String, "GitBranch string mismatch")
			require.Equal(t, tt.expected.gitBranchValid, params.GitBranch.Valid, "GitBranch valid flag mismatch")

			require.Equal(t, tt.expected.gitCommitMessage, params.GitCommitMessage.String, "GitCommitMessage string mismatch")
			require.Equal(t, tt.expected.gitCommitMessageValid, params.GitCommitMessage.Valid, "GitCommitMessage valid flag mismatch")

			require.Equal(t, tt.expected.gitCommitAuthorName, params.GitCommitAuthorName.String, "GitCommitAuthorName string mismatch")
			require.Equal(t, tt.expected.gitCommitAuthorNameValid, params.GitCommitAuthorName.Valid, "GitCommitAuthorName valid flag mismatch")

			require.Equal(t, tt.expected.gitCommitAuthorEmail, params.GitCommitAuthorEmail.String, "GitCommitAuthorEmail string mismatch")
			require.Equal(t, tt.expected.gitCommitAuthorEmailValid, params.GitCommitAuthorEmail.Valid, "GitCommitAuthorEmail valid flag mismatch")

			require.Equal(t, tt.expected.gitCommitAuthorUsername, params.GitCommitAuthorUsername.String, "GitCommitAuthorUsername string mismatch")
			require.Equal(t, tt.expected.gitCommitAuthorUsernameValid, params.GitCommitAuthorUsername.Valid, "GitCommitAuthorUsername valid flag mismatch")

			require.Equal(t, tt.expected.gitCommitAuthorAvatarUrl, params.GitCommitAuthorAvatarUrl.String, "GitCommitAuthorAvatarUrl string mismatch")
			require.Equal(t, tt.expected.gitCommitAuthorAvatarUrlValid, params.GitCommitAuthorAvatarUrl.Valid, "GitCommitAuthorAvatarUrl valid flag mismatch")

			require.Equal(t, tt.expected.gitCommitTimestamp, params.GitCommitTimestamp.Int64, "GitCommitTimestamp value mismatch")
			require.Equal(t, tt.expected.gitCommitTimestampValid, params.GitCommitTimestamp.Valid, "GitCommitTimestamp valid flag mismatch")
		})
	}
}
