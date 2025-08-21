package db

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

func TestWithRetry_Success(t *testing.T) {
	callCount := 0
	
	result, err := WithRetry(func() (string, error) {
		callCount++
		return "success", nil
	})
	
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, callCount, "should succeed on first try")
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
	
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount, "should retry twice then succeed")
}

func TestWithRetry_SkipsRetryOnNotFound(t *testing.T) {
	callCount := 0
	
	result, err := WithRetry(func() (string, error) {
		callCount++
		return "", sql.ErrNoRows
	})
	
	assert.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.Equal(t, "", result)
	assert.Equal(t, 1, callCount, "should not retry on not found error")
}

func TestWithRetry_SkipsRetryOnDuplicateKey(t *testing.T) {
	callCount := 0
	duplicateKeyErr := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
	
	result, err := WithRetry(func() (string, error) {
		callCount++
		return "", duplicateKeyErr
	})
	
	assert.Error(t, err)
	assert.True(t, IsDuplicateKeyError(err))
	assert.Equal(t, "", result)
	assert.Equal(t, 1, callCount, "should not retry on duplicate key error")
}

func TestWithRetry_ExhaustsRetries(t *testing.T) {
	callCount := 0
	transientErr := errors.New("persistent connection failure")
	
	result, err := WithRetry(func() (string, error) {
		callCount++
		return "", transientErr
	})
	
	assert.Error(t, err)
	assert.Equal(t, transientErr, err)
	assert.Equal(t, "", result)
	assert.Equal(t, 3, callCount, "should try 3 times then give up")
}

func TestWithRetry_GenericTypes(t *testing.T) {
	t.Run("int type", func(t *testing.T) {
		result, err := WithRetry(func() (int, error) {
			return 42, nil
		})
		
		assert.NoError(t, err)
		assert.Equal(t, 42, result)
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
		
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}

