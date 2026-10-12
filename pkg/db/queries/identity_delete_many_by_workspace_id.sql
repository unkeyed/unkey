-- name: DeleteManyIdentitiesByWorkspaceID :exec
-- DeleteManyIdentitiesByWorkspaceID hard deletes every identity in a
-- workspace. Intended for tests that reset reusable fixtures.
DELETE FROM identities
WHERE workspace_id = sqlc.arg(workspace_id);
