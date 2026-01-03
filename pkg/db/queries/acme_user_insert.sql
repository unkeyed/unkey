-- name: InsertAcmeUser :exec

INSERT INTO acme_users (id, workspace_id, encrypted_key, created_at)
VALUES (?,?,?,?);
