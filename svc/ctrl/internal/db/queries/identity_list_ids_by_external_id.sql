-- name: ListIdentityIDsByExternalID :many
-- Lists every identity of an end user, soft-deleted ones included, so a GDPR
-- erasure can find the end user's rows in tables keyed by identity id.
SELECT id FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
  AND external_id = sqlc.arg(external_id);
