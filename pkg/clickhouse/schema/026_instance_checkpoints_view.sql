-- Regular view that applies FINAL once so callers can't forget it. Prefer
-- `instance_checkpoints` unless an ordered window must preserve the physical
-- table's sort metadata; those queries can use `instance_checkpoints_v1 FINAL`.
--
-- ClickHouse pushes simple WHERE filters through regular views, so:
--   SELECT ... FROM instance_checkpoints WHERE ts BETWEEN ? AND ?
-- is as efficient as writing the FINAL directly.
--
-- The view exists because forgetting FINAL produces duplicate rows on any
-- un-merged insert batches, which in turn makes memory pair-integration and
-- per-container disk math overcount. Writing it once here removes the foot
-- from in front of the gun.
CREATE VIEW instance_checkpoints AS
SELECT *
FROM instance_checkpoints_v1
FINAL;
