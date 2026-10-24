-- name: ListRows :many
-- ListRows returns every row of a name. The counter itself only ever needs the
-- sum; this exists for tests and diagnostics, which need to see how a counter is
-- spread across rows.
SELECT writer_id, count, revision, delete_after, compaction_after, compaction_level
FROM counters
WHERE name = sqlc.arg(name)
ORDER BY writer_id;
