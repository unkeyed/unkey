-- name: FindSentinelsByVersions :many
-- FindSentinelsByVersions returns sentinels for specific versions.
-- Used after ListClusterStateVersions to hydrate the full sentinel data.
SELECT * FROM `sentinels` WHERE version IN (sqlc.slice(versions));
