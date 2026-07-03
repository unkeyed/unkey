-- name: FindCustomDomainByDomainOrWildcard :one
SELECT * FROM custom_domains
WHERE domain IN (?, ?)
ORDER BY
    CASE WHEN domain = ? THEN 0 ELSE 1 END
LIMIT 1;
