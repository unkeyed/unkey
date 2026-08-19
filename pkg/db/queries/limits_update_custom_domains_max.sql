-- name: UpdateCustomDomainsMax :exec
UPDATE `limits`
SET custom_domains_max = sqlc.arg(custom_domains_max)
WHERE workspace_id = sqlc.arg(workspace_id);
