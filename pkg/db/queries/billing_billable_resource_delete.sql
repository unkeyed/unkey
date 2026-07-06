-- name: DeleteBillingBillableResource :exec
-- Disables a keyspace/namespace for billing (removes it from the billable set).
DELETE FROM billing_billable_resources
WHERE workspace_id = sqlc.arg(workspace_id)
  AND resource_type = sqlc.arg(resource_type)
  AND resource_id = sqlc.arg(resource_id);
