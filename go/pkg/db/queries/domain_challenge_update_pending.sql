-- name: UpdateDomainChallengePending :exec
UPDATE domain_challenges
SET status = ?, token = ?, authorization = ?, updated_at = ?
WHERE domain_id = ?;
