-- name: InsertAcmeChallenge :exec
-- InsertAcmeChallenge appends a new ACME challenge row for a custom domain.
--
-- Certificate bootstrap writes initial rows with status "waiting" so the
-- background certificate worker can claim and progress them asynchronously.
-- This query intentionally does not upsert; duplicate challenge creation should
-- fail loudly so callers do not silently overwrite in-flight challenges.
INSERT INTO acme_challenges (
  workspace_id,
  domain_id,
  token,
  authorization,
  status,
  challenge_type,
  created_at,
  updated_at,
  expires_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
