-- name: FindCustomDomainWithCertByDomain :one
SELECT
    cd.*,
    c.id AS certificate_id
FROM custom_domains cd
LEFT JOIN certificates c ON c.hostname = cd.domain
WHERE cd.domain = sqlc.arg(domain);
