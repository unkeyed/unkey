-- name: ListKeysForApi :many
-- Lists keys for a specific API's keyAuth
-- Returns: keys, permissions, and total count
WITH key_results AS (
    SELECT
        k.*,
        COUNT(*) OVER() AS total_count
    FROM
        "keys" k
    WHERE
        k."keyAuthId" = $1
        AND k."deletedAtM" IS NULL
        AND k."workspaceId" = $4
        AND ($3 = '' OR k."identityId" = $3)
        AND ($2 = '' OR k."id" > $2)
    ORDER BY
        k."id"
    LIMIT $5
)
SELECT
    kr.*,
    COALESCE(kr.total_count, 0) AS total_count
FROM
    key_results kr;