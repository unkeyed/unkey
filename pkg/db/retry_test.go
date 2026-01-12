package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestWithRetryContext_Success(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	result, err := WithRetryContext(ctx, func() (string, error) {
		callCount++
		return "success", nil
	})

	require.NoError(t, err)
	require.Equal(t, "success", result)
	require.Equal(t, 1, callCount, "should succeed on first try")
}

func TestWithRetryContext_RetriesTransientErrors(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	// Use actual MySQL deadlock error (1213) which is recognized as transient
	deadlockErr := &mysql.MySQLError{Number: 1213, Message: "Deadlock found when trying to get lock"}

	result, err := WithRetryContext(ctx, func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", deadlockErr
		}
		return "success", nil
	})

	require.NoError(t, err)
	require.Equal(t, "success", result)
	require.Equal(t, 3, callCount, "should retry twice then succeed")
}

func TestWithRetryContext_SkipsRetryOnNotFound(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	result, err := WithRetryContext(ctx, func() (string, error) {
		callCount++
		return "", sql.ErrNoRows
	})

	require.Error(t, err)
	require.True(t, IsNotFound(err))
	require.Equal(t, "", result)
	require.Equal(t, 1, callCount, "should not retry on not found error")
}

func TestWithRetryContext_SkipsRetryOnDuplicateKey(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	duplicateKeyErr := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}

	result, err := WithRetryContext(ctx, func() (string, error) {
		callCount++
		return "", duplicateKeyErr
	})

	require.Error(t, err)
	require.True(t, IsDuplicateKeyError(err))
	require.Equal(t, "", result)
	require.Equal(t, 1, callCount, "should not retry on duplicate key error")
}

func TestWithRetryContext_ExhaustsRetries(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	// Use actual MySQL lock wait timeout error (1205) which is recognized as transient
	lockWaitErr := &mysql.MySQLError{Number: 1205, Message: "Lock wait timeout exceeded"}

	result, err := WithRetryContext(ctx, func() (string, error) {
		callCount++
		return "", lockWaitErr
	})

	require.Error(t, err)
	require.ErrorIs(t, err, lockWaitErr)
	require.Equal(t, "", result)
	require.Equal(t, 3, callCount, "should try 3 times then give up")
}

func TestWithRetryContext_GenericTypes(t *testing.T) {
	ctx := context.Background()

	t.Run("int type", func(t *testing.T) {
		result, err := WithRetryContext(ctx, func() (int, error) {
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
		result, err := WithRetryContext(ctx, func() (TestStruct, error) {
			return expected, nil
		})

		require.NoError(t, err)
		require.Equal(t, expected, result)
	})
}

func TestWithRetryContext_ContextCancellation(t *testing.T) {
	t.Run("context already cancelled before first attempt", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		callCount := 0
		result, err := WithRetryContext(ctx, func() (string, error) {
			callCount++
			return "should not be called", nil
		})

		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, "", result)
		require.Equal(t, 0, callCount, "should not call function when context already cancelled")
	})

	t.Run("context cancelled during backoff", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		time.AfterFunc(25*time.Millisecond, cancel)

		callCount := 0
		// Use actual MySQL deadlock error to trigger retry
		deadlockErr := &mysql.MySQLError{Number: 1213, Message: "Deadlock found"}
		result, err := WithRetryContext(ctx, func() (string, error) {
			callCount++
			return "", deadlockErr
		})

		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, "", result)
		require.Equal(t, 1, callCount, "should stop after first attempt when cancelled during backoff")
	})

	t.Run("context deadline exceeded during retry", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancel()

		callCount := 0
		// Use actual MySQL deadlock error to trigger retry
		deadlockErr := &mysql.MySQLError{Number: 1213, Message: "Deadlock found"}
		result, err := WithRetryContext(ctx, func() (string, error) {
			callCount++
			return "", deadlockErr
		})

		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, "", result)
		require.Equal(t, 1, callCount, "should stop after first attempt when deadline exceeded during backoff")
	})

	t.Run("success with valid context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		callCount := 0
		// Use actual MySQL deadlock error to trigger retry
		deadlockErr := &mysql.MySQLError{Number: 1213, Message: "Deadlock found"}
		result, err := WithRetryContext(ctx, func() (string, error) {
			callCount++
			if callCount < 2 {
				return "", deadlockErr
			}
			return "success", nil
		})

		require.NoError(t, err)
		require.Equal(t, "success", result)
		require.Equal(t, 2, callCount, "should retry and succeed with valid context")
	})
}

func TestWithRetryContext_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Set up test database using containers
	mysqlCfg := containers.MySQL(t)

	// Create database instance
	dbInstance, err := New(Config{
		PrimaryDSN: mysqlCfg.FormatDSN(),
		Logger:     logging.NewNoop(),
	})
	require.NoError(t, err)
	defer dbInstance.Close()

	// Create test data using sqlc statements
	workspaceID := uid.New(uid.WorkspacePrefix)
	keySpaceID := uid.New(uid.KeySpacePrefix)

	// Insert workspace using sqlc
	err = Query.InsertWorkspace(ctx, dbInstance.RW(), InsertWorkspaceParams{
		ID:        workspaceID,
		OrgID:     workspaceID,
		Name:      "Test Workspace",
		Slug:      uid.New("slug"),
		CreatedAt: time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Insert key space using sqlc
	err = Query.InsertKeySpace(ctx, dbInstance.RW(), InsertKeySpaceParams{
		ID:          keySpaceID,
		WorkspaceID: workspaceID,
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	t.Run("retry with real database - success after transient failure", func(t *testing.T) {
		callCount := 0
		// Use actual MySQL deadlock error to trigger retry
		deadlockErr := &mysql.MySQLError{Number: 1213, Message: "Deadlock found"}
		keyID, err := WithRetryContext(ctx, func() (string, error) {
			callCount++

			// Simulate transient failure on first attempt
			if callCount == 1 {
				return "", deadlockErr
			}

			// Succeed on second attempt - insert using sqlc
			keyID := uid.New(uid.KeyPrefix)
			err := Query.InsertKey(ctx, dbInstance.RW(), InsertKeyParams{
				ID:                keyID,
				KeySpaceID:        keySpaceID,
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
		require.NotEmpty(t, keyID)
		require.Equal(t, 2, callCount, "should retry once then succeed")
	})

	t.Run("retry with real database - no retry on duplicate key", func(t *testing.T) {
		// Insert initial key using sqlc
		keyID := uid.New(uid.KeyPrefix)

		keyParams := InsertKeyParams{
			ID:                keyID,
			KeySpaceID:        keySpaceID,
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
		_, err = WithRetryContext(ctx, func() (string, error) {
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
		_, err := WithRetryContext(ctx, func() (FindKeyForVerificationRow, error) {
			callCount++
			// Try to find non-existent key using sqlc - should not be retried
			return Query.FindKeyForVerification(ctx, dbInstance.RO(), uid.New(uid.KeyPrefix))
		})

		require.Error(t, err)
		require.True(t, IsNotFound(err))
		require.Equal(t, 1, callCount, "should not retry on not found error")
	})

	t.Run("context cancelled stops database operation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Cancel after 25ms - during first backoff (50ms)
		time.AfterFunc(25*time.Millisecond, cancel)

		callCount := 0
		insertedKeyID := ""
		// Use actual MySQL deadlock error to trigger retry
		deadlockErr := &mysql.MySQLError{Number: 1213, Message: "Deadlock found"}

		_, err := WithRetryContext(ctx, func() (string, error) {
			callCount++

			// Simulate transient error that would normally trigger retry
			if callCount == 1 {
				return "", deadlockErr
			}

			// This should never execute because context is cancelled during backoff
			keyID := uid.New(uid.KeyPrefix)
			err := Query.InsertKey(ctx, dbInstance.RW(), InsertKeyParams{
				ID:                keyID,
				KeySpaceID:        keySpaceID,
				Hash:              hash.Sha256(keyID),
				Start:             "cancelled_key",
				WorkspaceID:       workspaceID,
				ForWorkspaceID:    sql.NullString{},
				Name:              sql.NullString{String: "should_not_insert", Valid: true},
				IdentityID:        sql.NullString{},
				Meta:              sql.NullString{},
				Expires:           sql.NullTime{},
				CreatedAtM:        time.Now().UnixMilli(),
				Enabled:           true,
				RemainingRequests: sql.NullInt32{},
				RefillDay:         sql.NullInt16{},
				RefillAmount:      sql.NullInt32{},
			})
			if err == nil {
				insertedKeyID = keyID
			}
			return keyID, err
		})

		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 1, callCount, "should stop after first attempt")
		require.Empty(t, insertedKeyID, "should not insert key after context cancelled")
	})
}
