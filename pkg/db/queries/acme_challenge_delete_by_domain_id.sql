-- name: DeleteAcmeChallengeByDomainID :exec
DELETE FROM acme_challenges WHERE domain_id = sqlc.arg(domain_id);
