-- name: InsertComputeSignupCreditClaim :exec
-- Inserts a signup credit claim. Used only by tests; the dashboard owns claims.
INSERT INTO compute_signup_credit_claims (card_fingerprint, workspace_id, stripe_customer_id, workos_user_id, attempt_id, created_at)
VALUES (sqlc.arg(card_fingerprint), sqlc.arg(workspace_id), sqlc.arg(stripe_customer_id), sqlc.arg(workos_user_id), sqlc.arg(attempt_id), sqlc.arg(created_at));
