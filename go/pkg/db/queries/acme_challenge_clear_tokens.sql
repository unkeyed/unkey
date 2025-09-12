-- name: ClearAcmeChallengeTokens :exec
UPDATE acme_challenges
SET token = ?, authorization = ?, updated_at = ?
WHERE domain_id = ?;