-- name: FindDomainByDomain :one
SELECT * FROM domains WHERE domain = ?;
