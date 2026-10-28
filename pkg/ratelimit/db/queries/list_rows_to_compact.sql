-- name: ListRowsToCompact :many
-- ListRowsToCompact pages through rows whose compact_after_ns is before
-- before_ns, in the order of the compact_after_ns index. A page resumes after
-- the last row of the previous page. For the first page, pass an
-- after_compact_after_ns of -1; the after_fingerprint then does not matter. A
-- page can hold several rows of one fingerprint, so callers remove duplicates
-- and then handle the whole fingerprint with ListClosedRows.
SELECT fingerprint, compact_after_ns
FROM ratelimit_state
WHERE compact_after_ns < sqlc.arg(before_ns)
    AND compact_after_ns >= sqlc.arg(after_compact_after_ns)
    AND (compact_after_ns > sqlc.arg(after_compact_after_ns) OR fingerprint > sqlc.arg(after_fingerprint))
ORDER BY compact_after_ns, fingerprint
LIMIT ?;
