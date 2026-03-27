-- name: FindPortalConfigByID :one
SELECT * FROM portal_configurations WHERE id = ?;
