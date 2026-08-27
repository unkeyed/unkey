-- RecordLogdrainSuccess advances the composite cursor only for the current
-- fencing token and a lease that is valid at database time. An expired lease
-- or a non-monotonic cursor changes no rows.
-- name: RecordLogdrainSuccess :execrows
UPDATE logdrain_state
SET committed_offset_inserted_at = sqlc.arg(committed_offset_inserted_at),
  committed_offset_event_id = sqlc.arg(committed_offset_event_id),
  consecutive_failures = 0,
  next_attempt_at = 0,
  last_error = NULL
WHERE logdrain_id = sqlc.arg(logdrain_id)
  AND fencing_token = sqlc.arg(fencing_token)
  AND lease_expires_at > CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
  AND (
    committed_offset_inserted_at < sqlc.arg(committed_offset_inserted_at)
    OR (
      committed_offset_inserted_at = sqlc.arg(committed_offset_inserted_at)
      AND committed_offset_event_id < sqlc.arg(committed_offset_event_id)
    )
  );
