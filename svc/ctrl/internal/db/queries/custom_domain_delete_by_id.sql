-- name: DeleteCustomDomainByID :exec
DELETE d, c, cert
FROM custom_domains d
LEFT JOIN acme_challenges c ON c.domain_id = d.id
LEFT JOIN certificates cert ON cert.hostname = d.domain AND cert.workspace_id = d.workspace_id
WHERE d.id = sqlc.arg(id);
