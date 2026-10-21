-- name: TightenFreeTierApiThrottle :exec
UPDATE `limits` l
SET l.api_requests_count_max_per_minute = sqlc.arg('requests_per_minute')
WHERE l.workspace_id = sqlc.arg('workspace_id')
  AND (l.api_requests_count_max_per_minute IS NULL OR l.api_requests_count_max_per_minute > sqlc.arg('requests_per_minute'))
  AND EXISTS (
    SELECT 1 FROM `workspace_billing` b
    WHERE b.workspace_id = l.workspace_id AND b.tier = 'Free'
  );
