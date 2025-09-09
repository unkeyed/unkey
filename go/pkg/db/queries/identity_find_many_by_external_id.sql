-- name: FindIdentitiesByExternalId :many
SELECT *
FROM identities
WHERE workspace_id = ? AND external_id IN (sqlc.slice('externalIds')) AND deleted = ?;
