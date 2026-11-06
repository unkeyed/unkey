// Package keyleak filters signed secret-scanning reports from GitHub and
// TruffleHog before enqueueing notifications. Each [Reporter] gets its own
// endpoint and verifier and shares one report shape and validation. A 200
// response means no keys matched or every matching key's notification was
// accepted by Restate, not that delivery completed. Database lookup failures
// return 503 and are not durably retried. TruffleHog reports ask the worker to
// disable each matched key before alerting the workspace. GitHub reports never
// disable keys.
package keyleak
