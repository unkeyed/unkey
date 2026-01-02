-- name: InsertAcmeChallenge :exec
INSERT INTO acme_challenges (
    workspace_id,
    domain_id,
    token,
    authorization,
    status,
    challenge_type,
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
    ?,
    ?
);
