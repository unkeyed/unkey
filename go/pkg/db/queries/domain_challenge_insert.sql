-- name: InsertDomainChallenge :exec
INSERT INTO domain_challenges (
    workspace_id,
    domain_id,
    token,
    authorization,
    status,
    created_at,
    updated_at,
    expires_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);
