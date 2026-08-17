-- Regular view that applies FINAL once so ordinary callers can't forget it.
-- Prefer `instance_checkpoints` unless a query depends on the physical table's
-- input order, as described below.
--
-- ClickHouse pushes simple WHERE filters through regular views, so:
--   SELECT ... FROM instance_checkpoints WHERE ts BETWEEN ? AND ?
-- prunes the same parts as writing FINAL directly. Order-sensitive queries are
-- the exception: some ClickHouse versions do not propagate the table's input
-- order through the view, so window functions may add a full sort even when
-- their order matches the primary key. Those queries should read
-- `instance_checkpoints_v1 FINAL` directly.
--
-- The view exists because forgetting FINAL produces duplicate rows on any
-- un-merged insert batches, which in turn makes memory pair-integration and
-- per-container disk math overcount. Writing it once here removes the foot
-- from in front of the gun.
CREATE VIEW instance_checkpoints AS
SELECT *
FROM instance_checkpoints_v1
FINAL;
