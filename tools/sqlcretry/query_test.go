package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetrySafe(t *testing.T) {
	for _, tt := range []struct {
		name  string
		query string
		safe  bool
	}{
		{name: "read", query: "SELECT id FROM users WHERE id = ?", safe: true},
		{name: "sqlc slice", query: "SELECT id FROM users WHERE id IN (/*SLICE:ids*/?)", safe: true},
		{name: "union", query: "SELECT id FROM users UNION ALL SELECT id FROM archived_users", safe: true},
		{name: "cte", query: "WITH matches AS (SELECT id FROM users) SELECT id FROM matches", safe: true},
		{name: "aggregates", query: "SELECT COALESCE(JSON_ARRAYAGG(JSON_OBJECT('id', id)), JSON_ARRAY()), COUNT(*), MAX(id) FROM users", safe: true},
		{name: "time", query: "SELECT FLOOR(UNIX_TIMESTAMP(NOW(3)) * 1000)", safe: true},
		{name: "date arithmetic", query: "SELECT DATE_ADD(NOW(), INTERVAL 30 DAY), CURRENT_TIMESTAMP(3) - INTERVAL 2 HOUR", safe: true},
		{name: "lock words in literal", query: "SELECT 'FOR UPDATE', 'GET_LOCK(1)'", safe: true},
		{name: "lock words in comment", query: "SELECT id FROM users /* FOR UPDATE */", safe: true},
		{name: "update", query: "UPDATE users SET enabled = true", safe: false},
		{name: "delete", query: "DELETE FROM users", safe: false},
		{name: "insert", query: "INSERT INTO users (id) VALUES (?)", safe: false},
		{name: "exclusive lock", query: "SELECT id FROM users FOR UPDATE", safe: false},
		{name: "shared lock", query: "SELECT id FROM users LOCK IN SHARE MODE", safe: false},
		{name: "nested lock", query: "SELECT id FROM users WHERE id IN (SELECT id FROM users FOR UPDATE)", safe: false},
		{name: "cte lock", query: "WITH matches AS (SELECT id FROM users FOR UPDATE) SELECT id FROM matches", safe: false},
		{name: "union lock", query: "(SELECT id FROM users FOR UPDATE) UNION ALL (SELECT id FROM archived_users FOR UPDATE)", safe: false},
		{name: "skip locked", query: "SELECT id FROM users FOR UPDATE SKIP LOCKED", safe: false},
		{name: "executable comment", query: "SELECT id FROM users /*!99999 FOR UPDATE */", safe: false},
		{name: "optimizer hint", query: "SELECT /*+ SET_VAR(sort_buffer_size = 16M) */ id FROM users", safe: false},
		{name: "advisory lock", query: "SELECT GET_LOCK('name', 1)", safe: false},
		{name: "release lock", query: "SELECT RELEASE_LOCK('name')", safe: false},
		{name: "assignment", query: "SELECT @value := 1", safe: false},
		{name: "session variable", query: "SELECT @value", safe: false},
		{name: "session result", query: "SELECT LAST_INSERT_ID()", safe: false},
		{name: "session mutation", query: "SELECT LAST_INSERT_ID(1)", safe: false},
		{name: "found rows", query: "SELECT SQL_CALC_FOUND_ROWS id FROM users", safe: false},
		{name: "file output", query: "SELECT id FROM users INTO OUTFILE '/tmp/users'", safe: false},
		{name: "unknown function", query: "SELECT send_email(id) FROM users", safe: false},
		{name: "qualified function", query: "SELECT custom.lower(id) FROM users", safe: false},
		{name: "sleep", query: "SELECT SLEEP(1)", safe: false},
		{name: "sequence", query: "SELECT NEXTVAL(sequence)", safe: false},
		{name: "replication wait", query: "SELECT WAIT_FOR_EXECUTED_GTID_SET('uuid:1', 1)", safe: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			safe, err := retrySafe(tt.query)
			require.NoError(t, err)
			require.Equal(t, tt.safe, safe)
		})
	}
}

func TestRetrySafeRejectsUnparseableSQL(t *testing.T) {
	for _, query := range []string{"SELECT FROM", "SELECT id FROM users; DELETE FROM users", "SELECT NEXT 1 VALUES FROM sequence"} {
		_, err := retrySafe(query)
		require.Error(t, err)
	}
}
