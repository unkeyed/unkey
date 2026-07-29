-- name: FindAcmeChallengeByDomain :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
-- Returns the challenge row for a domain, if one exists. domain_id is unique on
-- acme_challenges, so there is at most one. Used as the idempotency check for
-- infra certificate provisioning: once a challenge exists the renewal cron owns
-- issuance, so provisioning is a no-op.
SELECT ac.* FROM acme_challenges ac
JOIN custom_domains cd ON (ac.domain_id = cd.id COLLATE utf8mb4_0900_ai_ci AND ac.domain_id = cd.id COLLATE utf8mb4_0900_as_cs)
WHERE cd.domain = ?;
