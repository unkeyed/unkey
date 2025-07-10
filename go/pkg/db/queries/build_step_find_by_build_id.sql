-- name: FindVersionStepsByVersionId :many
SELECT 
    version_id,
    status,
    message,
    error_message,
    created_at
FROM version_steps 
WHERE version_id = ?
ORDER BY created_at ASC;