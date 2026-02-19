-- name: DeploymentExistsByCommitShaAppAndEnv :one
SELECT EXISTS(
    SELECT 1 FROM deployments
    WHERE git_commit_sha = sqlc.arg(git_commit_sha)
      AND app_id = sqlc.arg(app_id)
      AND environment_id = sqlc.arg(environment_id)
) AS `exists`;
