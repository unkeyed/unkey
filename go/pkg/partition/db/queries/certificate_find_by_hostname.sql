-- name: FindCertificateByHostname :one
SELECT * FROM certificates WHERE hostname = ?;
