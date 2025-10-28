-- name: FindRemainingKey :one
SELECT remaining_requests FROM `keys` WHERE id = ?;
