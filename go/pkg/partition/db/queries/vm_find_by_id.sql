-- name: FindVMById :one
SELECT * FROM vms WHERE id = ?;
