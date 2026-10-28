-- name: Now :one
-- Now returns the clock of the MySQL server in Unix milliseconds. Cleanup
-- deletes faded rows by it.
SELECT CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED) AS now_ms;
