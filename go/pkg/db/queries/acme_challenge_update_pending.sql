-- name: UpdateACmeChallengePending :exec
UPDATE acme_challenges
SET status = ?, token = ?, authorization = ?, updated_at = ?
WHERE domain_id = ?;
