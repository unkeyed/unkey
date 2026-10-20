-- name: DeleteNames :execrows
-- DeleteNames removes every row of the names. Deleting only some rows of a name
-- would lower its SUM(spent_ns) under a writer whose fresh start was derived
-- from it, so names are only ever deleted whole.
DELETE FROM ratelimit_gcra
WHERE name IN (sqlc.slice(names));
