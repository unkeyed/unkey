-- name: GetCertificateHealth :one
SELECT
    COUNT(CASE WHEN dc.status = 'failed' THEN 1 END) AS failed_challenges,
    CAST(COALESCE(MIN(CASE WHEN c.id IS NOT NULL AND dc.expires_at > 0 THEN dc.expires_at END), 0) AS SIGNED) AS earliest_expiry
FROM acme_challenges dc
JOIN custom_domains d ON dc.domain_id = d.id
LEFT JOIN certificates c ON c.hostname = d.domain;
