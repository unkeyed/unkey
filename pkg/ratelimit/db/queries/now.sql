-- name: Now :one
-- Now returns the clock of the MySQL server in Unix nanoseconds, with
-- microsecond precision. Cleanup deletes faded rows by it.
SELECT CAST(UNIX_TIMESTAMP(NOW(6)) * 1000000000 AS SIGNED) AS now_ns;
