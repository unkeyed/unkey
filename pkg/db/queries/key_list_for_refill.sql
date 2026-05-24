-- name: ListKeysForRefill :many
-- ListKeysForRefill returns keys that need their remaining_requests refilled.
-- Splits the refill_day conditions into three UNION ALL branches so each can
-- use the idx_keys_refill index (deleted_at_m, refill_day, pk) via equality
-- lookups instead of an OR scan. Per-branch LIMIT enables early termination.
-- The deferred join on pk avoids reading full rows until after the combined
-- result is cursor-paginated and limited.
-- All four LIMIT params must be set to the same batch size value.
SELECT k.pk, k.id, k.workspace_id, k.refill_amount, k.remaining_requests, k.name
FROM `keys` k
INNER JOIN (
    SELECT u.pk FROM (
        (
            SELECT /*+ INDEX(ki idx_keys_refill) */ ki.pk
            FROM `keys` ki
            WHERE ki.deleted_at_m IS NULL
              AND ki.refill_day IS NULL
              AND ki.refill_amount IS NOT NULL
              AND (ki.remaining_requests IS NULL OR ki.refill_amount > ki.remaining_requests)
              AND ki.pk > sqlc.arg(after_pk)
            ORDER BY ki.pk
            LIMIT ?
        )
        UNION ALL
        (
            SELECT /*+ INDEX(ki idx_keys_refill) */ ki.pk
            FROM `keys` ki
            WHERE ki.deleted_at_m IS NULL
              AND ki.refill_day = sqlc.arg(today_day)
              AND ki.refill_amount IS NOT NULL
              AND (ki.remaining_requests IS NULL OR ki.refill_amount > ki.remaining_requests)
              AND ki.pk > sqlc.arg(after_pk)
            ORDER BY ki.pk
            LIMIT ?
        )
        UNION ALL
        (
            SELECT /*+ INDEX(ki idx_keys_refill) */ ki.pk
            FROM `keys` ki
            WHERE ki.deleted_at_m IS NULL
              AND sqlc.arg(is_last_day_of_month) = 1
              AND ki.refill_day > sqlc.arg(today_day)
              AND ki.refill_amount IS NOT NULL
              AND (ki.remaining_requests IS NULL OR ki.refill_amount > ki.remaining_requests)
              AND ki.pk > sqlc.arg(after_pk)
            ORDER BY ki.pk
            LIMIT ?
        )
    ) AS u
    ORDER BY u.pk
    LIMIT ?
) AS batch ON batch.pk = k.pk;
