-- Lease lock so that, across multiple replicas, only one runs the reminder loop
-- per interval. A holder wins by inserting/updating the row only when the
-- current lease has expired; the lease is time-bounded so a crashed holder is
-- automatically released.
CREATE TABLE IF NOT EXISTS cron_locks (
    name         TEXT        NOT NULL PRIMARY KEY,
    holder       TEXT        NOT NULL,
    locked_until TIMESTAMPTZ NOT NULL
);
