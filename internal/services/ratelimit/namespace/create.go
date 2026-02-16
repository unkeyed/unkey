package namespace

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Create inserts a new namespace into the database. If another request races
// and creates it first (duplicate key), Create re-fetches from the DB.
// An audit log is always written for successful creation.
func (s *service) Create(ctx context.Context, workspaceID, name string, audit *AuditContext) (db.FindRatelimitNamespace, error) {
	ns, err := db.TxWithResultRetry(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) (db.FindRatelimitNamespace, error) {
		now := time.Now().UnixMilli()
		id := uid.New(uid.RatelimitNamespacePrefix)

		err := db.Query.InsertRatelimitNamespace(ctx, tx, db.InsertRatelimitNamespaceParams{
			ID:          id,
			WorkspaceID: workspaceID,
			Name:        name,
			CreatedAt:   now,
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return db.FindRatelimitNamespace{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Public("An unexpected error occurred while creating the namespace."),
			)
		}

		if db.IsDuplicateKeyError(err) {
			// Another request created it first — re-fetch using the write connection
			// to avoid read replica lag (fixes #4507)
			return s.loadFromDBWith(ctx, tx, workspaceID, name)
		}

		result := db.FindRatelimitNamespace{
			ID:                id,
			WorkspaceID:       workspaceID,
			Name:              name,
			CreatedAtM:        now,
			UpdatedAtM:        sql.NullInt64{Valid: false, Int64: 0},
			DeletedAtM:        sql.NullInt64{Valid: false, Int64: 0},
			DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
			WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
		}

		if audit != nil {
			auditErr := s.auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
				{
					WorkspaceID: workspaceID,
					Event:       auditlog.RatelimitNamespaceCreateEvent,
					Display:     "Created ratelimit namespace " + name,
					ActorID:     audit.ActorID,
					ActorName:   audit.ActorName,
					ActorMeta:   map[string]any{},
					ActorType:   audit.ActorType,
					RemoteIP:    audit.RemoteIP,
					UserAgent:   audit.UserAgent,
					Resources: []auditlog.AuditLogResource{
						{
							ID:          id,
							Type:        auditlog.RatelimitNamespaceResourceType,
							Meta:        nil,
							Name:        name,
							DisplayName: name,
						},
					},
				},
			})
			if auditErr != nil {
				return result, auditErr
			}
		}

		// Warm cache by both name and ID
		s.cache.Set(ctx, cache.ScopedKey{WorkspaceID: workspaceID, Key: name}, result)
		s.cache.Set(ctx, cache.ScopedKey{WorkspaceID: workspaceID, Key: id}, result)

		return result, nil
	})

	return ns, err
}

// CreateMany bulk-inserts namespaces into the database. Handles duplicate-key
// races by re-fetching any that were created concurrently.
func (s *service) CreateMany(ctx context.Context, workspaceID string, names []string, audit *AuditContext) (map[string]db.FindRatelimitNamespace, error) {
	created, err := db.TxWithResultRetry(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) (map[string]db.FindRatelimitNamespace, error) {
		now := time.Now().UnixMilli()
		result := make(map[string]db.FindRatelimitNamespace, len(names))

		insertParams := make([]db.InsertRatelimitNamespaceParams, len(names))
		auditLogs := make([]auditlog.AuditLog, len(names))
		nameToID := make(map[string]string, len(names))

		for i, name := range names {
			id := uid.New(uid.RatelimitNamespacePrefix)
			nameToID[name] = id

			insertParams[i] = db.InsertRatelimitNamespaceParams{
				ID:          id,
				WorkspaceID: workspaceID,
				Name:        name,
				CreatedAt:   now,
			}

			if audit != nil {
				auditLogs[i] = auditlog.AuditLog{
					WorkspaceID: workspaceID,
					Event:       auditlog.RatelimitNamespaceCreateEvent,
					Display:     "Created ratelimit namespace " + name,
					ActorID:     audit.ActorID,
					ActorName:   audit.ActorName,
					ActorMeta:   map[string]any{},
					ActorType:   audit.ActorType,
					RemoteIP:    audit.RemoteIP,
					UserAgent:   audit.UserAgent,
					Resources: []auditlog.AuditLogResource{
						{
							ID:          id,
							Type:        auditlog.RatelimitNamespaceResourceType,
							Meta:        nil,
							Name:        name,
							DisplayName: name,
						},
					},
				}
			}
		}

		err := db.BulkQuery.InsertRatelimitNamespaces(ctx, tx, insertParams)
		if err != nil && !db.IsDuplicateKeyError(err) {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Public("An unexpected error occurred while creating the namespaces."),
			)
		}

		if err == nil {
			// All inserts succeeded — build result map
			for _, name := range names {
				id := nameToID[name]
				result[name] = db.FindRatelimitNamespace{
					ID:                id,
					WorkspaceID:       workspaceID,
					Name:              name,
					CreatedAtM:        now,
					UpdatedAtM:        sql.NullInt64{Valid: false, Int64: 0},
					DeletedAtM:        sql.NullInt64{Valid: false, Int64: 0},
					DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
					WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
				}
			}

			if audit != nil && len(auditLogs) > 0 {
				auditErr := s.auditlogs.Insert(ctx, tx, auditLogs)
				if auditErr != nil {
					return nil, auditErr
				}
			}
		}
		// If duplicate key error, result stays empty — caller will re-fetch

		return result, nil
	})
	if err != nil {
		return nil, err
	}

	// For any names that were successfully created, warm the cache.
	// For any names not in the result (due to race), re-fetch from DB.
	for _, name := range names {
		if ns, ok := created[name]; ok {
			s.cache.Set(ctx, cache.ScopedKey{WorkspaceID: workspaceID, Key: name}, ns)
			s.cache.Set(ctx, cache.ScopedKey{WorkspaceID: workspaceID, Key: ns.ID}, ns)
		} else {
			// Race: re-fetch from DB
			ns, fetchErr := s.loadFromDB(ctx, workspaceID, name)
			if fetchErr != nil {
				return nil, fault.Wrap(fetchErr,
					fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Public("Failed to fetch namespace after race condition."),
				)
			}
			created[name] = ns
			s.cache.Set(ctx, cache.ScopedKey{WorkspaceID: workspaceID, Key: name}, ns)
			s.cache.Set(ctx, cache.ScopedKey{WorkspaceID: workspaceID, Key: ns.ID}, ns)
		}
	}

	return created, nil
}
