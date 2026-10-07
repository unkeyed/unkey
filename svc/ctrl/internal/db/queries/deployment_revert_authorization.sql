-- name: RevertDeploymentAuthorization :execresult
-- RevertDeploymentAuthorization puts a deployment back to awaiting_approval so
-- the approve button reappears. Call it when an approval was accepted but the
-- deployment never actually started.
--
-- Two conditions guard the update, and both live here rather than in Go
-- because either can change between a read and a write:
--
--   status = 'pending'     a cancel may have landed since the approval
--   invocation_id IS NULL  the run may already be out
--
-- The invocation id matters because Create sends the run to Restate first and
-- writes the id afterwards. An id on the row means a build is already going
-- out, and showing an approve button for it would be wrong.
UPDATE deployments
SET status = 'awaiting_approval', updated_at = sqlc.arg('updated_at')
WHERE id = sqlc.arg('id')
  AND status = 'pending'
  AND invocation_id IS NULL;
