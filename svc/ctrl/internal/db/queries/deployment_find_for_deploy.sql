-- name: FindDeploymentForDeploy :one
SELECT d.id, d.workspace_id, d.project_id, d.app_id, d.environment_id, d.status, d.created_at,
       d.cpu_millicores, d.memory_mib, d.storage_mib,
       d.git_commit_sha, d.git_branch, d.fork_repository_full_name, d.`trigger`,
       d.github_deployment_id, d.pr_number,
       w.slug AS workspace_slug, w.k8s_namespace AS workspace_k8s_namespace,
       p.slug AS project_slug,
       a.slug AS app_slug, a.is_rolled_back AS app_is_rolled_back,
       e.slug AS environment_slug, e.kind AS environment_kind
FROM deployments d
JOIN workspaces w ON w.id = d.workspace_id
JOIN projects p ON p.id = d.project_id
JOIN apps a ON a.id = d.app_id
JOIN environments e ON e.id = d.environment_id
WHERE d.id = sqlc.arg(id);
