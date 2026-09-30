-- RecordLogdrainSuccess advances the composite cursor and schedules the next
-- attempt after the supplied delay. It changes no rows for an expired lease,
-- stale fencing token, or non-monotonic cursor.
-- name: RecordLogdrainSuccess :execrows
UPDATE logdrains
SET committed_offset_inserted_at = sqlc.arg(committed_offset_inserted_at),
  committed_offset_event_id = sqlc.arg(committed_offset_event_id),
  consecutive_failures = 0,
  next_attempt_at = CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED) + CAST(sqlc.arg(next_attempt_delay_millis) AS SIGNED)
WHERE id = sqlc.arg(logdrain_id)
  AND fencing_token = sqlc.arg(fencing_token)
  AND lease_expires_at > CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
  AND (
    committed_offset_inserted_at < sqlc.arg(committed_offset_inserted_at)
    OR (
      committed_offset_inserted_at = sqlc.arg(committed_offset_inserted_at)
      AND committed_offset_event_id < sqlc.arg(committed_offset_event_id)
    )
  );
