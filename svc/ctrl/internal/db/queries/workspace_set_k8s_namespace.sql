-- name: SetWorkspaceK8sNamespace :exec
UPDATE `workspaces`
SET k8s_namespace = sqlc.arg(k8s_namespace)
WHERE id = sqlc.arg(id) AND k8s_namespace IS NULL;
;
