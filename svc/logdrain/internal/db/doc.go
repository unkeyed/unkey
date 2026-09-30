// Package db provides database access for log drain configuration, delivery
// state, and leases. Fencing tokens reject state writes from workers that no
// longer own a valid lease.
package db
