-- name: InsertAcmeUser :exec

INSERT INTO acme_users (workspace_id, encrypted_key)
VALUES (?,?);
