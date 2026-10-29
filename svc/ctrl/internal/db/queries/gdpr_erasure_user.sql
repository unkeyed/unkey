-- name: EraseUserCliSessions :execrows
DELETE FROM cli_sessions
WHERE user_id = sqlc.arg(user_id)
LIMIT 10000;

-- name: EraseUserDeviceAuthorizations :execrows
DELETE FROM device_authorizations
WHERE approver_user_id = sqlc.arg(user_id)
LIMIT 10000;

-- name: EraseUserComputeSignupCreditClaims :execrows
DELETE FROM compute_signup_credit_claims
WHERE workos_user_id = sqlc.arg(user_id)
LIMIT 10000;

-- name: PseudonymizeUserDeploymentTriggers :execrows
UPDATE deployments
SET triggered_by = NULL
WHERE triggered_by = sqlc.arg(user_id)
LIMIT 10000;

-- name: PseudonymizeUserAuditOutbox :execrows
UPDATE clickhouse_outbox
SET payload = JSON_REMOVE(payload, '$.actor.name', '$.actor.meta', '$.remote_ip', '$.user_agent')
WHERE version = 'audit_log.v1'
  AND JSON_EXTRACT(payload, '$.actor.id') = CAST(sqlc.arg(user_id) AS JSON)
  AND JSON_CONTAINS_PATH(payload, 'one', '$.actor.name', '$.actor.meta', '$.remote_ip', '$.user_agent')
LIMIT 10000;
