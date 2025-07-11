-- name: FindBranchByProjectName :one
SELECT 
    id,
    workspace_id,
    project_id,
    name,
    created_at,
    updated_at
FROM branches 
WHERE project_id = ? AND name = ?;