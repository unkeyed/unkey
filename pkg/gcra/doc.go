// Package gcra is a distributed rate limiter based on the Generic Cell Rate
// Algorithm, sharing state through MySQL.
//
// Every name has one row per writer. A writer is one process's entry for a
// name; it holds two values that only grow: spent_ns, the time it admitted,
// and fresh_start_ns, its highest proposed baseline. The state of a name is
// reset_at = MAX(fresh_start_ns) + SUM(spent_ns) over all writers, and a
// request fits if reset_at, moved by its cost, stays within one burst of now.
// See [Decide] for the decision itself.
//
// Decisions never wait for MySQL writes. A Limiter decides with its own values
// from memory and its peers' values from a replica, cached for about a second,
// and persists changed entries in batches about once a second. Writers that
// have not seen each other's spending yet can over-admit; because spending is
// summed, the excess delays later requests instead of being forgotten.
package gcra
