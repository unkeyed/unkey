-- name: InsertAcmeUser :execlastid

INSERT INTO acme_users (workspace_id, encrypted_key, created_at)
VALUES (?,?,?);
