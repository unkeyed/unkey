package mysql

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFindFirstString(t *testing.T) {
	value, ok := FindFirstString(
		sql.NullString{String: "ignored", Valid: false},
		sql.NullString{String: "first", Valid: true},
		sql.NullString{String: "second", Valid: true},
	)
	require.True(t, ok)
	require.Equal(t, "first", value)

	value, ok = FindFirstString(sql.NullString{String: "ignored", Valid: false})
	require.False(t, ok)
	require.Empty(t, value)

	value, ok = FindFirstString()
	require.False(t, ok)
	require.Empty(t, value)
}

func TestFindFirstBool(t *testing.T) {
	value, ok := FindFirstBool(
		sql.NullBool{Bool: true, Valid: false},
		sql.NullBool{Bool: false, Valid: true},
		sql.NullBool{Bool: true, Valid: true},
	)
	require.True(t, ok)
	require.False(t, value)

	value, ok = FindFirstBool(sql.NullBool{Bool: true, Valid: false})
	require.False(t, ok)
	require.False(t, value)

	value, ok = FindFirstBool()
	require.False(t, ok)
	require.False(t, value)
}

func TestFindFirstInt16(t *testing.T) {
	value, ok := FindFirstInt16(
		sql.NullInt16{Int16: -1, Valid: false},
		sql.NullInt16{Int16: 16, Valid: true},
		sql.NullInt16{Int16: 17, Valid: true},
	)
	require.True(t, ok)
	require.Equal(t, int16(16), value)

	value, ok = FindFirstInt16(sql.NullInt16{Int16: -1, Valid: false})
	require.False(t, ok)
	require.Zero(t, value)

	value, ok = FindFirstInt16()
	require.False(t, ok)
	require.Zero(t, value)
}

func TestFindFirstInt32(t *testing.T) {
	value, ok := FindFirstInt32(
		sql.NullInt32{Int32: -1, Valid: false},
		sql.NullInt32{Int32: 32, Valid: true},
		sql.NullInt32{Int32: 33, Valid: true},
	)
	require.True(t, ok)
	require.Equal(t, int32(32), value)

	value, ok = FindFirstInt32(sql.NullInt32{Int32: -1, Valid: false})
	require.False(t, ok)
	require.Zero(t, value)

	value, ok = FindFirstInt32()
	require.False(t, ok)
	require.Zero(t, value)
}

func TestFindFirstInt64(t *testing.T) {
	value, ok := FindFirstInt64(
		sql.NullInt64{Int64: -1, Valid: false},
		sql.NullInt64{Int64: 64, Valid: true},
		sql.NullInt64{Int64: 65, Valid: true},
	)
	require.True(t, ok)
	require.Equal(t, int64(64), value)

	value, ok = FindFirstInt64(sql.NullInt64{Int64: -1, Valid: false})
	require.False(t, ok)
	require.Zero(t, value)

	value, ok = FindFirstInt64()
	require.False(t, ok)
	require.Zero(t, value)
}

func TestFindFirstTime(t *testing.T) {
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC)
	value, ok := FindFirstTime(
		sql.NullTime{Time: now.Add(-time.Hour), Valid: false},
		sql.NullTime{Time: now, Valid: true},
		sql.NullTime{Time: now.Add(time.Hour), Valid: true},
	)
	require.True(t, ok)
	require.Equal(t, now, value)

	value, ok = FindFirstTime(sql.NullTime{Time: now, Valid: false})
	require.False(t, ok)
	require.Zero(t, value)

	value, ok = FindFirstTime()
	require.False(t, ok)
	require.Zero(t, value)
}
