-- name: CountAppSourceOciByAppId :one
-- CountAppSourceOciByAppId counts an app's OCI source rows. Only tests use this query.
SELECT COUNT(*)
FROM `app_source_oci`
WHERE app_id = sqlc.arg(app_id);
