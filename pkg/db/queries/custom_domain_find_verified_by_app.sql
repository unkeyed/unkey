-- name: FindVerifiedCustomDomainByAppID :one
SELECT * FROM custom_domains
WHERE app_id = sqlc.arg(app_id) AND verification_status = 'verified'
ORDER BY created_at ASC, id ASC
LIMIT 1;
