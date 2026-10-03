-- name: FindFlagBySlug :one
-- FindFlagBySlug locks the definition during enrollment so a concurrent policy
-- change cannot race the permission check and override write.
SELECT pk, id, slug, description, type, default_value, allow_opt_in, allow_opt_out
FROM flags WHERE slug = sqlc.arg(slug) FOR UPDATE;
