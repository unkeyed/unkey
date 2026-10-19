-- name: DeleteWriterCounters :execrows
-- DeleteWriterCounters removes one writer's rows for the given names. A writer
-- uses it when it forgets those entries, so a surviving row cannot hold a higher
-- revision than the writer's next snapshot of the same name.
DELETE FROM counters
WHERE writer_id = sqlc.arg(writer_id)
    AND name IN (sqlc.slice('names'));
