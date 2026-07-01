-- name: FindVerifiedCustomDomainByDomainExcludingWorkspace :one
SELECT * FROM custom_domains
WHERE domain = sqlc.arg(domain)
  AND workspace_id != sqlc.arg(workspace_id)
  AND verification_status = 'verified'
LIMIT 1;
