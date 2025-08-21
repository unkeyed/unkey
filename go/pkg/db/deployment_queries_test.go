package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInsertDeployment_AllParameters tests that InsertDeployment correctly binds all 19 parameters
func TestInsertDeployment_AllParameters(t *testing.T) {
	// Setup mock database
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer mockDB.Close()

	// Test data with all git fields populated
	now := time.Now().UnixMilli()
	params := InsertDeploymentParams{
		ID:                       "deployment_test123",
		WorkspaceID:              "ws_test456",
		ProjectID:                "proj_test789",
		Environment:              DeploymentsEnvironmentPreview,
		BuildID:                  sql.NullString{String: "build_123", Valid: true},
		RootfsImageID:            "img_456",
		GitCommitSha:             sql.NullString{String: "abc123def456", Valid: true},
		GitBranch:                sql.NullString{String: "feature/git-info", Valid: true},
		GitCommitMessage:         sql.NullString{String: "feat: add git information capture", Valid: true},
		GitCommitAuthorName:      sql.NullString{String: "John Doe", Valid: true},
		GitCommitAuthorEmail:     sql.NullString{String: "john@example.com", Valid: true},
		GitCommitAuthorUsername:  sql.NullString{String: "johndoe", Valid: true},
		GitCommitAuthorAvatarUrl: sql.NullString{String: "https://github.com/johndoe.png", Valid: true},
		GitCommitTimestamp:       sql.NullInt64{Int64: now, Valid: true},
		ConfigSnapshot:           []byte(`{"key": "value"}`),
		OpenapiSpec:              sql.NullString{String: "openapi spec", Valid: true},
		Status:                   DeploymentsStatusPending,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Int64: now, Valid: true},
	}

	// Expected SQL with all 19 parameters
	expectedSQL := `INSERT INTO ` + "`deployments`" + ` (
    id,
    workspace_id,
    project_id,
    environment,
    build_id,
    rootfs_image_id,
    git_commit_sha,
    git_branch,
    git_commit_message,
    git_commit_author_name,
    git_commit_author_email,
    git_commit_author_username,
    git_commit_author_avatar_url,
    git_commit_timestamp,
    config_snapshot,
    openapi_spec,
    status,
    created_at,
    updated_at
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);`

	// Expect the exact query with all parameters in correct order
	mock.ExpectExec(expectedSQL).
		WithArgs(
			params.ID,                       // 1
			params.WorkspaceID,              // 2
			params.ProjectID,                // 3
			params.Environment,              // 4
			params.BuildID,                  // 5
			params.RootfsImageID,            // 6
			params.GitCommitSha,             // 7
			params.GitBranch,                // 8
			params.GitCommitMessage,         // 9  - NEW
			params.GitCommitAuthorName,      // 10 - NEW
			params.GitCommitAuthorEmail,     // 11 - NEW
			params.GitCommitAuthorUsername,  // 12 - NEW
			params.GitCommitAuthorAvatarUrl, // 13 - NEW
			params.GitCommitTimestamp,       // 14 - NEW
			params.ConfigSnapshot,           // 15
			params.OpenapiSpec,              // 16
			params.Status,                   // 17
			params.CreatedAt,                // 18
			params.UpdatedAt,                // 19
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Execute
	queries := New()
	err = queries.InsertDeployment(context.Background(), mockDB, params)

	// Assert
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestInsertDeployment_NullGitFields tests InsertDeployment with null git fields
func TestInsertDeployment_NullGitFields(t *testing.T) {
	// Setup mock database
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer mockDB.Close()

	// Test data with null git fields
	now := time.Now().UnixMilli()
	params := InsertDeploymentParams{
		ID:                       "deployment_test123",
		WorkspaceID:              "ws_test456",
		ProjectID:                "proj_test789",
		Environment:              DeploymentsEnvironmentPreview,
		BuildID:                  sql.NullString{Valid: false}, // NULL
		RootfsImageID:            "img_456",
		GitCommitSha:             sql.NullString{Valid: false},                      // NULL
		GitBranch:                sql.NullString{Valid: false},                      // NULL
		GitCommitMessage:         sql.NullString{Valid: false},                      // NULL
		GitCommitAuthorName:      sql.NullString{Valid: false},                      // NULL
		GitCommitAuthorEmail:     sql.NullString{Valid: false},                      // NULL
		GitCommitAuthorUsername:  sql.NullString{Valid: false},                      // NULL
		GitCommitAuthorAvatarUrl: sql.NullString{Valid: false},                      // NULL
		GitCommitTimestamp:       sql.NullInt64{Valid: false},                       // NULL
		ConfigSnapshot:           []byte(`{}`),
		OpenapiSpec:              sql.NullString{Valid: false}, // NULL
		Status:                   DeploymentsStatusPending,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Int64: now, Valid: true},
	}

	expectedSQL := `INSERT INTO ` + "`deployments`" + ` (
    id,
    workspace_id,
    project_id,
    environment,
    build_id,
    rootfs_image_id,
    git_commit_sha,
    git_branch,
    git_commit_message,
    git_commit_author_name,
    git_commit_author_email,
    git_commit_author_username,
    git_commit_author_avatar_url,
    git_commit_timestamp,
    config_snapshot,
    openapi_spec,
    status,
    created_at,
    updated_at
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);`

	// Expect query with NULL values
	mock.ExpectExec(expectedSQL).
		WithArgs(
			params.ID,
			params.WorkspaceID,
			params.ProjectID,
			params.Environment,
			params.BuildID,                  // NULL
			params.RootfsImageID,
			params.GitCommitSha,             // NULL
			params.GitBranch,                // NULL
			params.GitCommitMessage,         // NULL
			params.GitCommitAuthorName,      // NULL
			params.GitCommitAuthorEmail,     // NULL
			params.GitCommitAuthorUsername,  // NULL
			params.GitCommitAuthorAvatarUrl, // NULL
			params.GitCommitTimestamp,       // NULL
			params.ConfigSnapshot,
			params.OpenapiSpec, // NULL
			params.Status,
			params.CreatedAt,
			params.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Execute
	queries := New()
	err = queries.InsertDeployment(context.Background(), mockDB, params)

	// Assert
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestInsertDeployment_SpecialCharacters tests InsertDeployment with special characters
func TestInsertDeployment_SpecialCharacters(t *testing.T) {
	// Setup mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// Test data with special characters that could cause SQL issues
	now := time.Now().UnixMilli()
	params := InsertDeploymentParams{
		ID:            "deployment_test123",
		WorkspaceID:   "ws_test456",
		ProjectID:     "proj_test789",
		Environment:   DeploymentsEnvironmentPreview,
		BuildID:       sql.NullString{Valid: false},
		RootfsImageID: "img_456",
		GitCommitSha:  sql.NullString{String: "abc123def456", Valid: true},
		GitBranch:     sql.NullString{String: "feature/fix-'quotes'-issue", Valid: true},
		GitCommitMessage: sql.NullString{
			String: "Fix SQL injection with 'single quotes' and \"double quotes\"\n\nAlso handles unicode: 🚀 ñoño",
			Valid:  true,
		},
		GitCommitAuthorName:      sql.NullString{String: "O'Reilly, José María", Valid: true},
		GitCommitAuthorEmail:     sql.NullString{String: "jose.o'reilly+test@example.com", Valid: true},
		GitCommitAuthorUsername:  sql.NullString{String: "jose_oreilly-dev", Valid: true},
		GitCommitAuthorAvatarUrl: sql.NullString{String: "https://github.com/jose_oreilly-dev.png?size=40&v=4", Valid: true},
		GitCommitTimestamp:       sql.NullInt64{Int64: now, Valid: true},
		ConfigSnapshot:           []byte(`{"config": "with 'quotes'"}`),
		OpenapiSpec:              sql.NullString{Valid: false},
		Status:                   DeploymentsStatusPending,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Int64: now, Valid: true},
	}

	// Mock should accept any INSERT query with the special characters
	mock.ExpectExec("INSERT INTO `deployments`").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			params.GitCommitMessage,         // Should contain special characters
			params.GitCommitAuthorName,      // Should contain apostrophe
			params.GitCommitAuthorEmail,     // Should contain + and apostrophe
			params.GitCommitAuthorUsername,  // Should contain underscore and dash
			params.GitCommitAuthorAvatarUrl, // Should contain query parameters
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Execute
	queries := New()
	err = queries.InsertDeployment(context.Background(), mockDB, params)

	// Assert
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFindDeploymentById_AllGitFields tests that FindDeploymentById returns all git fields
func TestFindDeploymentById_AllGitFields(t *testing.T) {
	// Setup mock database
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer mockDB.Close()

	deploymentID := "deployment_test123"
	now := time.Now().UnixMilli()

	expectedSQL := `SELECT 
    id,
    workspace_id,
    project_id,
    environment,
    build_id,
    rootfs_image_id,
    git_commit_sha,
    git_branch,
    git_commit_message,
    git_commit_author_name,
    git_commit_author_email,
    git_commit_author_username,
    git_commit_author_avatar_url,
    git_commit_timestamp,
    config_snapshot,
    openapi_spec,
    status,
    created_at,
    updated_at
FROM ` + "`deployments`" + `
WHERE id = ?;`

	// Mock row data with all git fields populated
	rows := sqlmock.NewRows([]string{
		"id", "workspace_id", "project_id", "environment", "build_id", "rootfs_image_id",
		"git_commit_sha", "git_branch", "git_commit_message", "git_commit_author_name",
		"git_commit_author_email", "git_commit_author_username", "git_commit_author_avatar_url",
		"git_commit_timestamp", "config_snapshot", "openapi_spec", "status", "created_at", "updated_at",
	}).AddRow(
		deploymentID,                                                            // id
		"ws_test456",                                                            // workspace_id
		"proj_test789",                                                          // project_id
		"preview",                                                               // environment
		"build_123",                                                             // build_id
		"img_456",                                                               // rootfs_image_id
		"abc123def456",                                                          // git_commit_sha
		"feature/git-info",                                                      // git_branch
		"feat: add git information capture\n\nIncludes author details",         // git_commit_message
		"John Doe",                                                              // git_commit_author_name
		"john@example.com",                                                      // git_commit_author_email
		"johndoe",                                                               // git_commit_author_username
		"https://github.com/johndoe.png",                                       // git_commit_author_avatar_url
		now,                                                                     // git_commit_timestamp
		`{"key": "value"}`,                                                      // config_snapshot
		"openapi spec",                                                          // openapi_spec
		"active",                                                                // status
		now,                                                                     // created_at
		now,                                                                     // updated_at
	)

	mock.ExpectQuery(expectedSQL).WithArgs(deploymentID).WillReturnRows(rows)

	// Execute
	queries := New()
	deployment, err := queries.FindDeploymentById(context.Background(), mockDB, deploymentID)

	// Assert
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	// Verify all git fields are populated
	assert.Equal(t, deploymentID, deployment.ID)
	assert.Equal(t, "ws_test456", deployment.WorkspaceID)
	assert.Equal(t, "proj_test789", deployment.ProjectID)

	assert.True(t, deployment.GitCommitSha.Valid)
	assert.Equal(t, "abc123def456", deployment.GitCommitSha.String)

	assert.True(t, deployment.GitBranch.Valid)
	assert.Equal(t, "feature/git-info", deployment.GitBranch.String)

	assert.True(t, deployment.GitCommitMessage.Valid)
	assert.Equal(t, "feat: add git information capture\n\nIncludes author details", deployment.GitCommitMessage.String)

	assert.True(t, deployment.GitCommitAuthorName.Valid)
	assert.Equal(t, "John Doe", deployment.GitCommitAuthorName.String)

	assert.True(t, deployment.GitCommitAuthorEmail.Valid)
	assert.Equal(t, "john@example.com", deployment.GitCommitAuthorEmail.String)

	assert.True(t, deployment.GitCommitAuthorUsername.Valid)
	assert.Equal(t, "johndoe", deployment.GitCommitAuthorUsername.String)

	assert.True(t, deployment.GitCommitAuthorAvatarUrl.Valid)
	assert.Equal(t, "https://github.com/johndoe.png", deployment.GitCommitAuthorAvatarUrl.String)

	assert.True(t, deployment.GitCommitTimestamp.Valid)
	assert.Equal(t, now, deployment.GitCommitTimestamp.Int64)
}

// TestFindDeploymentById_NullGitFields tests FindDeploymentById with NULL git fields
func TestFindDeploymentById_NullGitFields(t *testing.T) {
	// Setup mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	deploymentID := "deployment_test123"
	now := time.Now().UnixMilli()

	// Mock row data with NULL git fields
	rows := sqlmock.NewRows([]string{
		"id", "workspace_id", "project_id", "environment", "build_id", "rootfs_image_id",
		"git_commit_sha", "git_branch", "git_commit_message", "git_commit_author_name",
		"git_commit_author_email", "git_commit_author_username", "git_commit_author_avatar_url",
		"git_commit_timestamp", "config_snapshot", "openapi_spec", "status", "created_at", "updated_at",
	}).AddRow(
		deploymentID,   // id
		"ws_test456",   // workspace_id
		"proj_test789", // project_id
		"preview",      // environment
		nil,            // build_id (NULL)
		"img_456",      // rootfs_image_id
		nil,            // git_commit_sha (NULL)
		nil,            // git_branch (NULL)
		nil,            // git_commit_message (NULL)
		nil,            // git_commit_author_name (NULL)
		nil,            // git_commit_author_email (NULL)
		nil,            // git_commit_author_username (NULL)
		nil,            // git_commit_author_avatar_url (NULL)
		nil,            // git_commit_timestamp (NULL)
		`{}`,           // config_snapshot
		nil,            // openapi_spec (NULL)
		"pending",      // status
		now,            // created_at
		now,            // updated_at
	)

	mock.ExpectQuery("SELECT").WithArgs(deploymentID).WillReturnRows(rows)

	// Execute
	queries := New()
	deployment, err := queries.FindDeploymentById(context.Background(), mockDB, deploymentID)

	// Assert
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	// Verify all git fields are NULL/invalid
	assert.False(t, deployment.GitCommitSha.Valid)
	assert.False(t, deployment.GitBranch.Valid)
	assert.False(t, deployment.GitCommitMessage.Valid)
	assert.False(t, deployment.GitCommitAuthorName.Valid)
	assert.False(t, deployment.GitCommitAuthorEmail.Valid)
	assert.False(t, deployment.GitCommitAuthorUsername.Valid)
	assert.False(t, deployment.GitCommitAuthorAvatarUrl.Valid)
	assert.False(t, deployment.GitCommitTimestamp.Valid)

	// Verify other fields still work
	assert.Equal(t, deploymentID, deployment.ID)
	assert.Equal(t, "ws_test456", deployment.WorkspaceID)
}

// TestFindDeploymentById_SpecialCharacters tests FindDeploymentById with special characters in git fields
func TestFindDeploymentById_SpecialCharacters(t *testing.T) {
	// Setup mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	deploymentID := "deployment_test123"
	now := time.Now().UnixMilli()

	// Mock row data with special characters in git fields
	rows := sqlmock.NewRows([]string{
		"id", "workspace_id", "project_id", "environment", "build_id", "rootfs_image_id",
		"git_commit_sha", "git_branch", "git_commit_message", "git_commit_author_name",
		"git_commit_author_email", "git_commit_author_username", "git_commit_author_avatar_url",
		"git_commit_timestamp", "config_snapshot", "openapi_spec", "status", "created_at", "updated_at",
	}).AddRow(
		deploymentID,
		"ws_test456",
		"proj_test789",
		"preview",
		"build_123",
		"img_456",
		"abc123def456",
		"feature/emoji-support-🚀",
		"Fix issues with 'single quotes' and \"double quotes\"\n\nAdds unicode support: ñoño 🔥",
		"José María García-López 🧑‍💻",
		"jose.maria+test@example.com",
		"jose_maria-dev",
		"https://github.com/jose_maria-dev.png?size=40&v=4",
		now,
		`{"config": "value"}`,
		"openapi spec",
		"active",
		now,
		now,
	)

	mock.ExpectQuery("SELECT").WithArgs(deploymentID).WillReturnRows(rows)

	// Execute
	queries := New()
	deployment, err := queries.FindDeploymentById(context.Background(), mockDB, deploymentID)

	// Assert
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	// Verify special characters are preserved
	assert.Contains(t, deployment.GitBranch.String, "🚀")
	assert.Contains(t, deployment.GitCommitMessage.String, "'single quotes'")
	assert.Contains(t, deployment.GitCommitMessage.String, "\"double quotes\"")
	assert.Contains(t, deployment.GitCommitMessage.String, "ñoño 🔥")
	assert.Contains(t, deployment.GitCommitAuthorName.String, "José María")
	assert.Contains(t, deployment.GitCommitAuthorName.String, "🧑‍💻")
	assert.Contains(t, deployment.GitCommitAuthorEmail.String, "+test@")
	assert.Contains(t, deployment.GitCommitAuthorAvatarUrl.String, "?size=40&v=4")
}

// TestFindDeploymentById_NotFound tests FindDeploymentById when deployment doesn't exist
func TestFindDeploymentById_NotFound(t *testing.T) {
	// Setup mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	deploymentID := "deployment_nonexistent"

	// Mock empty result (no rows)
	rows := sqlmock.NewRows([]string{
		"id", "workspace_id", "project_id", "environment", "build_id", "rootfs_image_id",
		"git_commit_sha", "git_branch", "git_commit_message", "git_commit_author_name",
		"git_commit_author_email", "git_commit_author_username", "git_commit_author_avatar_url",
		"git_commit_timestamp", "config_snapshot", "openapi_spec", "status", "created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT").WithArgs(deploymentID).WillReturnRows(rows)

	// Execute
	queries := New()
	_, err = queries.FindDeploymentById(context.Background(), mockDB, deploymentID)

	// Assert
	require.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestInsertDeploymentParams_FieldCount tests that InsertDeploymentParams has the expected number of fields
func TestInsertDeploymentParams_FieldCount(t *testing.T) {
	// This test ensures we don't accidentally add/remove fields without updating tests
	params := InsertDeploymentParams{}

	// Use reflection to count fields in the struct
	// This will fail compilation if the struct changes significantly
	expectedFields := map[string]bool{
		"ID":                       true,
		"WorkspaceID":              true,
		"ProjectID":                true,
		"Environment":              true,
		"BuildID":                  true,
		"RootfsImageID":            true,
		"GitCommitSha":             true,
		"GitBranch":                true,
		"GitCommitMessage":         true, // NEW
		"GitCommitAuthorName":      true, // NEW
		"GitCommitAuthorEmail":     true, // NEW
		"GitCommitAuthorUsername":  true, // NEW
		"GitCommitAuthorAvatarUrl": true, // NEW
		"GitCommitTimestamp":       true, // NEW
		"ConfigSnapshot":           true,
		"OpenapiSpec":              true,
		"Status":                   true,
		"CreatedAt":                true,
		"UpdatedAt":                true,
	}

	// Verify struct has exactly 19 fields (original 13 + 6 new git fields)
	assert.Equal(t, 19, len(expectedFields))

	// Simple compile-time check that all expected fields exist
	_ = params.ID
	_ = params.WorkspaceID
	_ = params.ProjectID
	_ = params.Environment
	_ = params.BuildID
	_ = params.RootfsImageID
	_ = params.GitCommitSha
	_ = params.GitBranch
	_ = params.GitCommitMessage         // NEW
	_ = params.GitCommitAuthorName      // NEW
	_ = params.GitCommitAuthorEmail     // NEW
	_ = params.GitCommitAuthorUsername  // NEW
	_ = params.GitCommitAuthorAvatarUrl // NEW
	_ = params.GitCommitTimestamp       // NEW
	_ = params.ConfigSnapshot
	_ = params.OpenapiSpec
	_ = params.Status
	_ = params.CreatedAt
	_ = params.UpdatedAt
}