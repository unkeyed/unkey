-- name: ListBillingBillableResources :many
-- All resources a workspace has enabled for end-user billing. Presence of a row
-- means the keyspace/namespace is billable; absence means excluded.
SELECT *
FROM billing_billable_resources
WHERE workspace_id = sqlc.arg(workspace_id)
ORDER BY resource_type, resource_id;
