package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestWithRetry_Success(t *testing.T) {
	callCount := 0

	result, err := WithRetry(func() (string, error) {
		callCount++
		return "success", nil
	})

	require.NoError(t, err)
	require.Equal(t, "success", result)
	require.Equal(t, 1, callCount, "should succeed on first try")
}

func TestWithRetry_RetriesTransientErrors(t *testing.T) {
	callCount := 0
	transientErr := errors.New("connection timeout")

	result, err := WithRetry(func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", transientErr
		}
		return "success", nil
	})

	require.NoError(t, err)
	require.Equal(t, "success", result)
	require.Equal(t, 3, callCount, "should retry twice then succeed")
}

func TestWithRetry_SkipsRetryOnNotFound(t *testing.T) {
	callCount := 0

	result, err := WithRetry(func() (string, error) {
		callCount++
		return "", sql.ErrNoRows
	})

	require.Error(t, err)
	require.True(t, IsNotFound(err))
	require.Equal(t, "", result)
	require.Equal(t, 1, callCount, "should not retry on not found error")
}

func TestWithRetry_SkipsRetryOnDuplicateKey(t *testing.T) {
	callCount := 0
	duplicateKeyErr := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}

	result, err := WithRetry(func() (string, error) {
		callCount++
		return "", duplicateKeyErr
	})

	require.Error(t, err)
	require.True(t, IsDuplicateKeyError(err))
	require.Equal(t, "", result)
	require.Equal(t, 1, callCount, "should not retry on duplicate key error")
}

func TestWithRetry_ExhaustsRetries(t *testing.T) {
	callCount := 0
	transientErr := errors.New("persistent connection failure")

	result, err := WithRetry(func() (string, error) {
		callCount++
		return "", transientErr
	})

	require.Error(t, err)
	require.Equal(t, transientErr, err)
	require.Equal(t, "", result)
	require.Equal(t, 3, callCount, "should try 3 times then give up")
}

func TestWithRetry_GenericTypes(t *testing.T) {
	t.Run("int type", func(t *testing.T) {
		result, err := WithRetry(func() (int, error) {
			return 42, nil
		})

		require.NoError(t, err)
		require.Equal(t, 42, result)
	})

	t.Run("struct type", func(t *testing.T) {
		type TestStruct struct {
			ID   int
			Name string
		}

		expected := TestStruct{ID: 1, Name: "test"}
		result, err := WithRetry(func() (TestStruct, error) {
			return expected, nil
		})

		require.NoError(t, err)
		require.Equal(t, expected, result)
	})
}

// TestWithRetry_Integration tests retry functionality with a real database connection
// This test requires Docker to be running for the MySQL container
func TestWithRetry_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx := context.Background()

	// Set up test database using containers
	mysqlCfg := containers.MySQL(t)
	mysqlCfg.DBName = "unkey"

	// Create database instance
	dbInstance, err := New(Config{
		PrimaryDSN: mysqlCfg.FormatDSN(),
		Logger:     logging.NewNoop(),
	})
	require.NoError(t, err)
	defer dbInstance.Close()

	// Create test data using sqlc statements
	workspaceID := uid.New(uid.WorkspacePrefix)
	keyringID := uid.New(uid.KeyAuthPrefix)

	// Insert workspace using sqlc
	err = Query.InsertWorkspace(ctx, dbInstance.RW(), InsertWorkspaceParams{
		ID:        workspaceID,
		OrgID:     workspaceID,
		Name:      "Test Workspace",
		Slug:      uid.New("slug"),
		CreatedAt: time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Insert keyring using sqlc
	err = Query.InsertKeyring(ctx, dbInstance.RW(), InsertKeyringParams{
		ID:          keyringID,
		WorkspaceID: workspaceID,
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	t.Run("retry with real database - success after transient failure", func(t *testing.T) {
		callCount := 0
		_, err := WithRetry(func() (string, error) {
			callCount++

			// Simulate transient failure on first attempt
			if callCount == 1 {
				return "", errors.New("dial tcp: connection refused")
			}

			// Succeed on second attempt - insert using sqlc
			keyID := uid.New(uid.KeyPrefix)
			err := Query.InsertKey(ctx, dbInstance.RW(), InsertKeyParams{
				ID:                keyID,
				KeyringID:         keyringID,
				Hash:              hash.Sha256(keyID),
				Start:             "retry_start",
				WorkspaceID:       workspaceID,
				ForWorkspaceID:    sql.NullString{},
				Name:              sql.NullString{String: "retry_key", Valid: true},
				IdentityID:        sql.NullString{},
				Meta:              sql.NullString{},
				Expires:           sql.NullTime{},
				CreatedAtM:        time.Now().UnixMilli(),
				Enabled:           true,
				RemainingRequests: sql.NullInt32{},
				RefillDay:         sql.NullInt16{},
				RefillAmount:      sql.NullInt32{},
			})

			return keyID, err
		})

		require.NoError(t, err)
		require.Equal(t, 2, callCount, "should retry once then succeed")
	})

	t.Run("retry with real database - no retry on duplicate key", func(t *testing.T) {
		// Insert initial key using sqlc
		keyID := uid.New(uid.KeyPrefix)

		keyParams := InsertKeyParams{
			ID:                keyID,
			KeyringID:         keyringID,
			Hash:              hash.Sha256(keyID),
			Start:             "dup_start",
			WorkspaceID:       workspaceID,
			ForWorkspaceID:    sql.NullString{},
			Name:              sql.NullString{String: "dup_key", Valid: true},
			IdentityID:        sql.NullString{},
			Meta:              sql.NullString{},
			Expires:           sql.NullTime{},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
			RemainingRequests: sql.NullInt32{},
			RefillDay:         sql.NullInt16{},
			RefillAmount:      sql.NullInt32{},
		}
		err := Query.InsertKey(ctx, dbInstance.RW(), keyParams)
		require.NoError(t, err)

		callCount := 0
		_, err = WithRetry(func() (string, error) {
			callCount++

			// Try to insert duplicate key - should not be retried
			err = Query.InsertKey(ctx, dbInstance.RW(), keyParams)
			return "success", err
		})

		require.Error(t, err)
		require.True(t, IsDuplicateKeyError(err))
		require.Equal(t, 1, callCount, "should not retry on duplicate key error")
	})

	t.Run("retry with real database - no retry on not found", func(t *testing.T) {
		callCount := 0
		_, err := WithRetry(func() (FindKeyForVerificationRow, error) {
			callCount++
			// Try to find non-existent key using sqlc - should not be retried
			return Query.FindKeyForVerification(ctx, dbInstance.RO(), uid.New(uid.KeyPrefix))
		})

		require.Error(t, err)
		require.True(t, IsNotFound(err))
		require.Equal(t, 1, callCount, "should not retry on not found error")
	})
}
