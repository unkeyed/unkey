package namespace

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
)

// AuditContext provides the actor and request information needed for audit logging
// when creating namespaces. This is passed by the caller so that permission checks
// can stay in the handler layer between Get and Create calls.
type AuditContext struct {
	ActorID   string
	ActorName string
	ActorType auditlog.AuditLogActor
	RemoteIP  string
	UserAgent string
}

// Service provides namespace lookup, creation, and cache management.
//
// Operations are split into Get + Create (rather than FindOrCreate) so that
// callers can perform permission checks between the two calls.
type Service interface {
	// Get looks up a namespace by name or ID within the given workspace.
	// Returns the namespace, whether it was found, and any error.
	// A not-found result is (zero, false, nil) — not an error.
	Get(ctx context.Context, workspaceID, nameOrID string) (db.FindRatelimitNamespace, bool, error)

	// GetMany looks up multiple namespaces by name within the given workspace.
	// Returns a map of found namespaces keyed by name, a slice of missing names,
	// and any error.
	GetMany(ctx context.Context, workspaceID string, names []string) (found map[string]db.FindRatelimitNamespace, missing []string, err error)

	// Create inserts a new namespace into the database. If another request
	// races and creates it first (duplicate key), Create re-fetches from the DB.
	// An audit log is written when audit is non-nil.
	Create(ctx context.Context, workspaceID, name string, audit *AuditContext) (db.FindRatelimitNamespace, error)

	// CreateMany bulk-inserts namespaces into the database. Handles duplicate-key
	// races by re-fetching any that were created concurrently.
	// Returns a map of all created (or re-fetched) namespaces keyed by name.
	CreateMany(ctx context.Context, workspaceID string, names []string, audit *AuditContext) (map[string]db.FindRatelimitNamespace, error)

	// Invalidate removes the namespace from cache by both name and ID.
	Invalidate(ctx context.Context, workspaceID string, ns db.FindRatelimitNamespace)
}
