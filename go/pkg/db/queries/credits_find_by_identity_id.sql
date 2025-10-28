-- name: FindCreditsByIdentityID :one
SELECT * FROM `credits` WHERE identity_id = ?;
