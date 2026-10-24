-- name: DeleteMergedRows :execrows
-- DeleteMergedRows removes the given writers' rows of one name, once compaction
-- has merged them into a compacted row.
DELETE FROM counters
WHERE name = sqlc.arg(name)
    AND writer_id IN (sqlc.slice('writer_ids'));
