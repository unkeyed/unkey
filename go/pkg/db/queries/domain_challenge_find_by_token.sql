-- name: FindDomainChallengeByToken :one
SELECT * FROM domain_challenges WHERE workspace_id = ? AND domain_id = ? AND token = ?;
