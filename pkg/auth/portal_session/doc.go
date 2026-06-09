// Package portalsession authenticates customer portal browser sessions.
//
// The resolver only claims cookie-only requests. Requests with an Authorization
// header are left for bearer-token resolvers so a stale portal cookie cannot
// override an explicit root key or JWT.
package portalsession
