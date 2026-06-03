-- name: ListDueDeletions :many
-- Returns deletions whose grace window has elapsed, oldest first, up
-- to limit rows. Used by the permanent-delete cron sweep to fan out
-- per-resource hard-delete VOs.
SELECT *
FROM `deletions`
WHERE delete_permanently_at <= ?
ORDER BY delete_permanently_at ASC
LIMIT ?;
