-- name: UpsertBillingBillableResource :exec
-- Enables a keyspace/namespace for billing. Idempotent: the unique
-- (workspace_id, resource_type, resource_id) index makes a repeat enable a
-- no-op rather than a duplicate row.
INSERT IGNORE INTO billing_billable_resources (
    id,
    workspace_id,
    resource_type,
    resource_id,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(resource_type),
    sqlc.arg(resource_id),
    sqlc.arg(created_at)
);
