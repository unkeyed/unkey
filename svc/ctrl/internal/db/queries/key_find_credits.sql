-- name: FindKeyCredits :one
SELECT remaining_requests FROM `keys` k WHERE k.id = ?;
