-- name: UpdateWorkspaceDeleteProtection :exec
-- Sets a workspace's delete protection. Used only by tests: the dashboard
-- creates every workspace protected, and a GDPR workspace erasure refuses a
-- protected one.
UPDATE `workspaces` SET delete_protection = sqlc.arg(delete_protection) WHERE id = sqlc.arg(id);
