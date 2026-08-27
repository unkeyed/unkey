// Package db provides database access for log drain configuration, delivery
// state, and leases. Protobuf configuration with inline encrypted secrets lives
// in logdrains. Mutable delivery state and lease ownership live in
// logdrain_state. Fencing tokens reject state writes from workers that no longer
// own a valid lease.
package db
