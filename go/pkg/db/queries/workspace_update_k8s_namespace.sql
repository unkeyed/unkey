-- name: UpdateWorkspaceK8sNamespace :execresult
UPDATE `workspaces`
SET k8s_namespace = sqlc.arg(k8s_namespace)
WHERE id = sqlc.arg(id) and k8s_namespace is null
;
