-- name: DeleteRegionsWithClusters :exec
-- Removes the given regions and their clusters. Integration tests that commit
-- regions to the shared database call this; production never deletes a region.
DELETE r, c
FROM regions r
LEFT JOIN clusters c ON c.region_id = r.id
WHERE r.id IN (sqlc.slice('ids'));
