-- name: FindCertificatesByHostnames :many
SELECT * FROM certificates WHERE hostname IN (sqlc.slice('hostnames'));
