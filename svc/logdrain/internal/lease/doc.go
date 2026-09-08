// Package lease assigns log drains to service processes without coupling lease
// ownership to polling or delivery.
//
// The service acquires expired leases and refreshes the leases that it acquired.
// A unique startup lease ID routes leases to the poller in the same process.
// Each acquisition writes a new fencing token. Delivery state writes must
// match that token, so a worker cannot mutate state after another acquisition.
//
// Lease queries use database time for expiry checks and compute each new
// absolute expiry from database time plus a supplied TTL. Refresh scheduling
// and database delay must stay below the 60-second minimum refresh margin.
//
// Fencing protects database state, but it cannot cancel an external delivery
// that was already in progress. Delivery is therefore at-least-once across
// lease expiry and node failure.
package lease
