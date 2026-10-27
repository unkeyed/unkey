-- name: DeploymentProjectOwnsImage :one
SELECT EXISTS (
    SELECT 1
    FROM deployments
    WHERE project_id = sqlc.arg(project_id)
      AND image_resolved = sqlc.arg(image_resolved)
    LIMIT 1
) AS owns_image;
