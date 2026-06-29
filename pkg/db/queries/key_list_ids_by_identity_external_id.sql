-- name: ListKeyIDsByIdentityExternalID :many
-- Deleted keys are intentionally included: their verification events are
-- immutable history that still belongs to this identity, and the root-key
-- analytics path (scoped by key_space_id) already surfaces them. The identity
-- is still required to be live (i.deleted = false) so a reused external_id
-- resolves to the current identity rather than a soft-deleted predecessor.
--
-- keys has no FK on identity_id, so we constrain k.workspace_id explicitly as
-- defense-in-depth rather than trusting the join to stay within the workspace.
SELECT k.id
FROM `keys` k
JOIN identities i ON k.identity_id = i.id
WHERE i.workspace_id = sqlc.arg(workspace_id)
  AND k.workspace_id = sqlc.arg(workspace_id)
  AND i.external_id = sqlc.arg(external_id)
  AND i.deleted = false;
