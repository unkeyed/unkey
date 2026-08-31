-- name: FindKeyCredits :one
SELECT remaining_requests FROM `keys` k WHERE k.id = ?;

-- name: FindLiveKeyCredits :one
SELECT remaining_requests
FROM `keys`
WHERE id = sqlc.arg(id)
  AND deleted_at_m IS NULL;
