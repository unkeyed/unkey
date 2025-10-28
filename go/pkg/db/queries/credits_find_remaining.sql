-- name: FindRemainingCredits :one
SELECT remaining FROM `credits` WHERE id = ?;
