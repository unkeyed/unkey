-- name: ListPrivateNetworkClusters :many
-- ListPrivateNetworkClusters supplies cell identities for DNS locality.
-- Clusters without a cell ID cannot identify imported EndpointSlices.
SELECT c.cell_id, r.platform, r.name AS region
FROM clusters c
INNER JOIN regions r ON r.id = c.region_id
WHERE r.platform = sqlc.arg(platform)
    AND c.cell_id IS NOT NULL
ORDER BY c.cell_id;
