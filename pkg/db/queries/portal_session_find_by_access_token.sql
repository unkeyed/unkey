-- name: FindPortalSessionByAccessTokenHash :one
-- Resolves an access token to its session, one indexed read against the UNIQUE
-- `idx_access_token_hash`.
--
-- Deliberately unfiltered by expiry or revocation: the row is cached, and both
-- of those are clock- or write-driven state the cache would pin to whatever was
-- true at fill time. The caller derives session state from the row against the
-- current clock instead.
SELECT pk, id, workspace_id, portal_id, external_id, scopes, preview, exchange_code_hash, exchange_code_expires_at, access_token_hash, access_token_created_at, access_token_expires_at, revoked_at, return_url, created_at FROM portal_sessions
WHERE access_token_hash = sqlc.arg(access_token_hash);
