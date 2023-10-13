-- name: InsertWorkspace :exec
INSERT INTO
    `workspaces` (
        id,
        name,
        slug,
        tenant_id,
        plan,
        features,
        beta_features
    )
VALUES
    (
        sqlc.arg("id"),
        sqlc.arg("name"),
        sqlc.arg("slug"),
        sqlc.arg("tenant_id"),
        sqlc.arg("plan"),
        sqlc.arg("features"),
        sqlc.arg("beta_features")
    );