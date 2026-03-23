-- name: FindPortalBrandingByConfigID :one
SELECT * FROM portal_branding WHERE portal_config_id = ?;
