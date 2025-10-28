-- name: FindCreditsByKeyID :one
SELECT * FROM `credits` WHERE key_id = ?;
