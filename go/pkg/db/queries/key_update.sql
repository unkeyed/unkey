-- name: UpdateKey :exec
UPDATE `keys` k SET
    name = CASE 
        WHEN CAST(sqlc.arg('name_specified') AS UNSIGNED) = 1 THEN sqlc.narg('name') 
        ELSE k.name 
    END,
    identity_id = CASE 
        WHEN CAST(sqlc.arg('identity_id_specified') AS UNSIGNED) = 1 THEN sqlc.narg('identity_id') 
        ELSE k.identity_id 
    END,
    enabled = CASE 
        WHEN CAST(sqlc.arg('enabled_specified') AS UNSIGNED) = 1 THEN sqlc.narg('enabled') 
        ELSE k.enabled 
    END,
    meta = CASE 
        WHEN CAST(sqlc.arg('meta_specified') AS UNSIGNED) = 1 THEN sqlc.narg('meta') 
        ELSE k.meta 
    END,
    expires = CASE 
        WHEN CAST(sqlc.arg('expires_specified') AS UNSIGNED) = 1 THEN sqlc.narg('expires') 
        ELSE k.expires 
    END,
    updated_at_m = sqlc.arg('now')
WHERE id = sqlc.arg('id');
