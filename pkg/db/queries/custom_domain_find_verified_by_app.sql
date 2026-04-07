-- name: FindVerifiedCustomDomainByAppID :one
SELECT * FROM custom_domains
WHERE app_id = sqlc.arg(app_id) AND verification_status = 'verified'
LIMIT 1;
