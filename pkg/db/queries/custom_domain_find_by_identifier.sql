-- name: FindCustomDomainByIdentifier :one
-- Identifier is either a domain id or a domain name. Each UNION branch hits its own
-- unique index (custom_domains_id_unique, unique_domain_workspace_idx) as a point
-- lookup; a single OR predicate would scan every domain in the workspace instead.
-- Names are stored lowercase, so the name half is lowered here rather than left to the
-- column's collation. The id half is compared as given: ids are case-sensitive.
(SELECT
    cd_by_id.id,
    cd_by_id.project_id,
    cd_by_id.app_id,
    cd_by_id.environment_id,
    cd_by_id.domain,
    cd_by_id.verification_status,
    cd_by_id.verification_token,
    cd_by_id.ownership_verified,
    cd_by_id.cname_verified,
    cd_by_id.target_cname,
    cd_by_id.verification_error,
    cd_by_id.last_checked_at,
    cd_by_id.created_at,
    cd_by_id.updated_at
FROM custom_domains cd_by_id
WHERE cd_by_id.id = sqlc.arg(domain) AND cd_by_id.workspace_id = sqlc.arg(workspace_id)
LIMIT 1)
UNION ALL
(SELECT
    cd_by_name.id,
    cd_by_name.project_id,
    cd_by_name.app_id,
    cd_by_name.environment_id,
    cd_by_name.domain,
    cd_by_name.verification_status,
    cd_by_name.verification_token,
    cd_by_name.ownership_verified,
    cd_by_name.cname_verified,
    cd_by_name.target_cname,
    cd_by_name.verification_error,
    cd_by_name.last_checked_at,
    cd_by_name.created_at,
    cd_by_name.updated_at
FROM custom_domains cd_by_name
WHERE cd_by_name.workspace_id = sqlc.arg(workspace_id) AND cd_by_name.domain = LOWER(sqlc.arg(domain))
LIMIT 1)
LIMIT 1;
