// Package auth normalizes API credentials into principals and checks RBAC permissions.
//
// A [Service] tries configured [Resolver] implementations in order. This lets
// API handlers authenticate root keys, portal sessions, and future credential
// sources through one request flow. [Principal] mirrors frontline's versioned
// subject/type/source shape while keeping API-local workspace and permission
// fields until the API can consume frontline principals directly.
package auth
