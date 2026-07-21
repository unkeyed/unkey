-- name: UpdateClickhouseWorkspaceSettingsProvisioningState :exec
UPDATE `clickhouse_workspace_settings`
SET
    provisioning_state = sqlc.arg(provisioning_state),
    updated_at = sqlc.arg(updated_at)
WHERE workspace_id = sqlc.arg(workspace_id);
