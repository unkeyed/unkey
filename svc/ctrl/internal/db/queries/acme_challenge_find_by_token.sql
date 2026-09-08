-- name: FindAcmeChallengeByToken :one
SELECT acme_challenges.pk, acme_challenges.domain_id, acme_challenges.workspace_id, acme_challenges.token, acme_challenges.challenge_type, acme_challenges.authorization, acme_challenges.status, acme_challenges.expires_at, acme_challenges.created_at, acme_challenges.updated_at FROM acme_challenges WHERE workspace_id = ? AND domain_id = ? AND token = ?;
