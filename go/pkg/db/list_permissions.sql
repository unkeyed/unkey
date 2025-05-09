-- name: ListPermissions :many
-- Lists permissions for a specific workspace with pagination support
-- Returns: permission records and total count
WITH permission_results AS (
    SELECT
        p.*,
        COUNT(*) OVER() AS total_count
    FROM
        "permissions" p
    WHERE
        p."workspaceId" = $1
        AND ($2 = '' OR p."id" > $2)
    ORDER BY
        p."id"
    LIMIT $3
)
SELECT
    pr.*,
    COALESCE(pr.total_count, 0) AS total_count
FROM
    permission_results pr;