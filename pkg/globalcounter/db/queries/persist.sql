-- name: Persist :exec
-- Persist uses an increasing revision to stop delayed writes or retries from
-- overwriting a newer snapshot, even when its count decreases.
--
-- Reordered delivery for one writer (time flows downward):
--
--         Writer                              MySQL (revision, count)
--     t0  persist (7, 100) --- [delayed] --.
--     t1  persist (8,  80) ----------------|--> (8, 80)
--     t2                                   `--> (8, 80), ignore revision 7
--
-- compaction_after moves to the database time of the write plus compact_delay_ms
-- whenever a newer revision is accepted, so compaction only merges rows that no
-- writer has touched for that long.
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
