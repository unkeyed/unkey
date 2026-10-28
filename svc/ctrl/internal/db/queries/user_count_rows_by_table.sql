-- name: CountUserRowsByTable :one
-- Counts a dashboard user's rows in every table a GDPR user erasure deletes.
-- Used only by tests.
SELECT
  (SELECT COUNT(*) FROM cli_sessions WHERE cli_sessions.user_id = sqlc.arg(user_id)) AS cli_sessions,
  (SELECT COUNT(*) FROM device_authorizations WHERE device_authorizations.approver_user_id = sqlc.arg(approver_user_id)) AS device_authorizations,
  (SELECT COUNT(*) FROM compute_signup_credit_claims WHERE compute_signup_credit_claims.workos_user_id = sqlc.arg(user_id)) AS compute_signup_credit_claims;
