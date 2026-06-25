-- name: UpdateDeploymentGitMetadata :exec
UPDATE deployments
SET
    git_commit_sha = ?,
    git_branch = ?,
    git_commit_message = ?,
    git_commit_author_handle = ?,
    git_commit_author_avatar_url = ?,
    git_commit_timestamp = ?,
    updated_at = ?
WHERE id = ?;
