-- name: ListExecutableChallenges :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT dc.workspace_id, dc.challenge_type, d.domain FROM acme_challenges dc
JOIN custom_domains d ON (dc.domain_id = d.id COLLATE utf8mb4_0900_ai_ci AND dc.domain_id = d.id COLLATE utf8mb4_0900_as_cs)
WHERE (dc.status = 'waiting' OR (dc.status = 'verified' AND dc.expires_at <= UNIX_TIMESTAMP(DATE_ADD(NOW(), INTERVAL 30 DAY)) * 1000))
AND dc.challenge_type IN (sqlc.slice(verification_types))
ORDER BY d.created_at ASC;
