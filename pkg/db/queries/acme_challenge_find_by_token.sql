-- name: FindAcmeChallengeByToken :one
SELECT * FROM acme_challenges WHERE workspace_id = ? AND domain_id = ? AND token = ?;
