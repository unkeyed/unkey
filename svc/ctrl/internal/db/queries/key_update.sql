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
    remaining_requests = CASE 
        WHEN CAST(sqlc.arg('remaining_requests_specified') AS UNSIGNED) = 1 THEN sqlc.narg('remaining_requests') 
        ELSE k.remaining_requests 
    END,
    refill_amount = CASE 
        WHEN CAST(sqlc.arg('refill_amount_specified') AS UNSIGNED) = 1 THEN sqlc.narg('refill_amount') 
        ELSE k.refill_amount 
    END,
    refill_day = CASE 
        WHEN CAST(sqlc.arg('refill_day_specified') AS UNSIGNED) = 1 THEN sqlc.narg('refill_day') 
        ELSE k.refill_day 
    END,
    updated_at_m = sqlc.arg('now')
WHERE id = sqlc.arg('id');
