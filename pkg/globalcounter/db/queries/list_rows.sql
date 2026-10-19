-- name: ListRows :many
-- ListRows returns every row of a name. Tests and diagnostics use it to see how
-- many rows a counter holds and what each one contains.
SELECT writer_id, count, revision, delete_after, compaction_after, compaction_level
FROM counters
WHERE name = sqlc.arg(name)
ORDER BY writer_id;
