-- name: Persist :exec
-- Persist writes a writer's full count, not a delta, so that a lost write is
-- repaired by the next one and a repeated write cannot count twice. That only
-- holds if a late or retried snapshot can never replace a newer one, and the
-- count cannot decide that, because a deduction lowers it. The revision decides
-- instead: only a newer revision changes the row.
--
-- Reordered delivery for one writer (time flows downward):
--
--         Writer                              MySQL (revision, count)
--     t0  persist (7, 100) --- [delayed] --.
--     t1  persist (8,  80) ----------------|--> (8, 80)
--     t2                                   `--> (8, 80), ignore revision 7
--
-- compaction_after uses the database clock, so pod clock skew cannot make a row
-- look idle early. It moves only when a newer revision is accepted, because a
-- rejected replay says nothing about whether the writer is still active.
--
-- Retry after a lost acknowledgement:
--
--         Writer                              MySQL (revision, count)
--     t0  persist (9, 50) -------------------> (9, 50)
--     t1                    X <--------------- acknowledgement lost
--     t2  retry   (9, 50) -------------------> (9, 50), unchanged
INSERT INTO counters (
    name,
    writer_id,
    count,
    revision,
    delete_after,
    compaction_after
) VALUES (
    sqlc.arg(name),
    sqlc.arg(writer_id),
    sqlc.arg(count),
    sqlc.arg(revision),
    sqlc.narg(delete_after),
    CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED) + sqlc.arg(compact_delay_ms)
)
ON DUPLICATE KEY UPDATE
    count = IF(VALUES(revision) > revision, VALUES(count), count),
    delete_after = IF(VALUES(revision) > revision, VALUES(delete_after), delete_after),
    compaction_after = IF(VALUES(revision) > revision, VALUES(compaction_after), compaction_after),
    revision = GREATEST(revision, VALUES(revision));
