package v2_ratelimit_multi_limit

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/apps/api/openapi"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/match"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
)

type (
	Request  = openapi.V2RatelimitMultiLimitRequestBody
	Response = openapi.V2RatelimitMultiLimitResponseBody
)

// Handler implements zen.Route interface for the v2 ratelimit multiLimit endpoint
type Handler struct {
	Logger                  logging.Logger
	Keys                    keys.KeyService
	DB                      db.Database
	ClickHouse              clickhouse.Bufferer
	Ratelimit               ratelimit.Service
	RatelimitNamespaceCache cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
	Auditlogs               auditlogs.AuditLogService
	TestMode                bool
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/ratelimit.multiLimit"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Collect unique namespaces and build cache keys
	uniqueNamespaces := make(map[string]bool)
	for _, check := range req {
		uniqueNamespaces[check.Namespace] = true
	}

	cacheKeys := make([]cache.ScopedKey, 0, len(uniqueNamespaces))
	for ns := range uniqueNamespaces {
		cacheKeys = append(cacheKeys, cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: ns})
	}

	// Batch load all namespaces using SWRMany
	namespaceLoader := func(ctx context.Context, keys []cache.ScopedKey) (map[cache.ScopedKey]db.FindRatelimitNamespace, error) {
		if len(keys) == 0 {
			return map[cache.ScopedKey]db.FindRatelimitNamespace{}, nil
		}

		results := make(map[cache.ScopedKey]db.FindRatelimitNamespace)
		namespaces := make([]string, 0, len(keys))
		for _, key := range keys {
			namespaces = append(namespaces, key.Key)
		}

		// Fetch all namespaces in a single batch query with overrides included
		rows, err := db.WithRetryContext(ctx, func() ([]db.FindManyRatelimitNamespacesRow, error) {
			return db.Query.FindManyRatelimitNamespaces(ctx, h.DB.RO(), db.FindManyRatelimitNamespacesParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Namespaces:  namespaces,
			})
		})
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			result := db.FindRatelimitNamespace{
				ID:                row.ID,
				WorkspaceID:       row.WorkspaceID,
				Name:              row.Name,
				CreatedAtM:        row.CreatedAtM,
				UpdatedAtM:        row.UpdatedAtM,
				DeletedAtM:        row.DeletedAtM,
				DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
				WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
			}

			overrides, err := db.UnmarshalNullableJSONTo[[]db.FindRatelimitNamespaceLimitOverride](row.Overrides)
			if err != nil {
				h.Logger.Error("failed to unmarshal overrides", "err", err)
			}

			for _, override := range overrides {
				result.DirectOverrides[override.Identifier] = override
				if strings.Contains(override.Identifier, "*") {
					result.WildcardOverrides = append(result.WildcardOverrides, override)
				}
			}

			// Cache by namespace name
			nameKey := cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: row.Name}
			results[nameKey] = result

			// Also cache by namespace ID to support ID-based lookups
			idKey := cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: row.ID}
			results[idKey] = result
		}

		return results, nil
	}

	namespaces, hits, err := h.RatelimitNamespaceCache.SWRMany(
		ctx,
		cacheKeys,
		namespaceLoader,
		caches.DefaultFindFirstOp,
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Public("An unexpected error occurred while fetching namespaces."),
		)
	}

	// Collect all missing namespaces
	missingKeys := make([]cache.ScopedKey, 0)
	for _, key := range cacheKeys {
		hit := hits[key]
		if hit == cache.Null {
			missingKeys = append(missingKeys, key)
		}
	}

	// Auto-create any missing namespaces
	if len(missingKeys) > 0 {
		err = h.createMissingNamespaces(ctx, s, auth, missingKeys, namespaces, namespaceLoader)
		if err != nil {
			return err
		}
	}

	// Verify permissions for rate limiting - user needs either wildcard OR ALL specific namespace permissions
	// Build a list of all specific namespace permissions from the request
	requiredPerms := make([]rbac.PermissionQuery, 0, len(req))
	for _, check := range req {
		cacheKey := cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: check.Namespace}
		namespace := namespaces[cacheKey]
		requiredPerms = append(requiredPerms, rbac.T(rbac.Tuple{
			ResourceType: rbac.Ratelimit,
			ResourceID:   namespace.ID,
			Action:       rbac.Limit,
		}))
	}

	// Create wildcard permission
	wildcardPermission := rbac.T(rbac.Tuple{
		ResourceType: rbac.Ratelimit,
		ResourceID:   "*",
		Action:       rbac.Limit,
	})

	// User needs EITHER wildcard permission OR ALL specific namespace permissions
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(wildcardPermission, rbac.And(requiredPerms...))))
	if err != nil {
		return err
	}

	// Build ratelimit requests for all checks
	ratelimitReqs := make([]ratelimit.RatelimitRequest, len(req))
	checkMetadata := make([]checkMeta, len(req))

	reqTime := time.Now()
	if h.TestMode {
		header := s.Request().Header.Get("X-Test-Time")
		if header != "" {
			if ts, parseErr := strconv.ParseInt(header, 10, 64); parseErr != nil {
				h.Logger.Warn("invalid test time", "header", header)
			} else {
				reqTime = time.UnixMilli(ts)
			}
		}
	}

	for i, check := range req {
		cacheKey := cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: check.Namespace}
		namespace := namespaces[cacheKey]

		if namespace.DeletedAtM.Valid {
			return fault.New("namespace was deleted",
				fault.Code(codes.Data.RatelimitNamespace.Gone.URN()),
				fault.Public("This namespace has been deleted. Contact support to restore."),
			)
		}

		// Apply override if found, otherwise use request values
		limit, duration, overrideID, err := getLimitAndDuration(check, namespace)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("error matching overrides"),
				fault.Public("Error matching ratelimit override"),
			)
		}

		ratelimitReqs[i] = ratelimit.RatelimitRequest{
			Name:       namespace.ID,
			Identifier: check.Identifier,
			Duration:   time.Duration(duration) * time.Millisecond,
			Limit:      limit,
			Cost:       ptr.SafeDeref(check.Cost, 1),
			Time:       reqTime,
		}

		checkMetadata[i] = checkMeta{
			namespaceName: check.Namespace,
			namespaceID:   namespace.ID,
			identifier:    check.Identifier,
			overrideID:    overrideID,
			limit:         limit,
		}
	}

	// Batch rate limit all requests using RatelimitMany
	start := time.Now()
	results, err := h.Ratelimit.RatelimitMany(ctx, ratelimitReqs)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("rate limit failed"),
			fault.Public("We're unable to process the rate limit requests."),
		)
	}
	latency := time.Since(start).Milliseconds()

	// Log to ClickHouse if enabled
	if s.ShouldLogRequestToClickHouse() {
		for i, result := range results {
			meta := checkMetadata[i]
			h.ClickHouse.BufferRatelimit(schema.Ratelimit{
				RequestID:   s.RequestID(),
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Time:        start.UnixMilli(),
				NamespaceID: meta.namespaceID,
				Identifier:  meta.identifier,
				Passed:      result.Success,
				Latency:     float64(latency) / float64(len(results)),
				OverrideID:  meta.overrideID,
				Limit:       uint64(result.Limit),
				Remaining:   uint64(result.Remaining),
				ResetAt:     result.Reset.UnixMilli(),
			})
		}
	}

	// Build response and calculate overall success
	limits := make([]openapi.V2RatelimitMultiLimitCheck, len(results))
	allPassed := true
	for i, result := range results {
		meta := checkMetadata[i]
		limits[i] = openapi.V2RatelimitMultiLimitCheck{
			Namespace:  meta.namespaceName,
			Identifier: meta.identifier,
			Passed:     result.Success,
			Limit:      meta.limit,
			Remaining:  result.Remaining,
			Reset:      result.Reset.UnixMilli(),
			OverrideId: meta.overrideID,
		}

		if !result.Success {
			allPassed = false
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2RatelimitMultiLimitResponseData{
			Passed: allPassed,
			Limits: limits,
		},
	})
}

func (h *Handler) createMissingNamespaces(
	ctx context.Context,
	s *zen.Session,
	auth *keys.KeyVerifier,
	missingKeys []cache.ScopedKey,
	namespaces map[cache.ScopedKey]db.FindRatelimitNamespace,
	namespaceLoader func(context.Context, []cache.ScopedKey) (map[cache.ScopedKey]db.FindRatelimitNamespace, error),
) error {
	// Verify permission to create namespace once for all missing namespaces
	err := auth.VerifyRootKey(ctx, keys.WithPermissions(
		rbac.T(
			rbac.Tuple{
				ResourceType: rbac.Ratelimit,
				ResourceID:   "*",
				Action:       rbac.CreateNamespace,
			},
		),
	))
	if err != nil {
		return err
	}

	createdNamespaces, err := db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (map[cache.ScopedKey]db.FindRatelimitNamespace, error) {
		now := time.Now().UnixMilli()
		created := make(map[cache.ScopedKey]db.FindRatelimitNamespace)

		// Prepare bulk insert params
		insertParams := make([]db.InsertRatelimitNamespaceParams, 0, len(missingKeys))
		auditLogs := make([]auditlog.AuditLog, 0, len(missingKeys))
		keyToID := make(map[cache.ScopedKey]string, len(missingKeys))

		for _, key := range missingKeys {
			id := uid.New(uid.RatelimitNamespacePrefix)
			keyToID[key] = id

			insertParams = append(insertParams, db.InsertRatelimitNamespaceParams{
				ID:          id,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Name:        key.Key,
				CreatedAt:   now,
			})

			// Collect audit log for this namespace
			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RatelimitNamespaceCreateEvent,
				Display:     "Created ratelimit namespace " + key.Key,
				ActorID:     auth.Key.ID,
				ActorName:   auth.Key.Name.String,
				ActorMeta:   map[string]any{},
				ActorType:   auditlog.RootKeyActor,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          id,
						Type:        auditlog.RatelimitNamespaceResourceType,
						Meta:        nil,
						Name:        key.Key,
						DisplayName: key.Key,
					},
				},
			})
		}

		// Bulk insert all namespaces in a single query
		err := db.BulkQuery.InsertRatelimitNamespaces(ctx, tx, insertParams)
		if err != nil && !db.IsDuplicateKeyError(err) {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Public("An unexpected error occurred while creating the namespaces."),
			)
		}

		// If successful (no race condition), build the created map
		if err == nil {
			for _, key := range missingKeys {
				id := keyToID[key]
				created[key] = db.FindRatelimitNamespace{
					ID:                id,
					WorkspaceID:       auth.AuthorizedWorkspaceID,
					Name:              key.Key,
					CreatedAtM:        now,
					UpdatedAtM:        sql.NullInt64{Valid: false, Int64: 0},
					DeletedAtM:        sql.NullInt64{Valid: false, Int64: 0},
					DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
					WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
				}
			}

			// Batch insert all audit logs
			if len(auditLogs) > 0 {
				err := h.Auditlogs.Insert(ctx, tx, auditLogs)
				if err != nil {
					return nil, err
				}
			}
		}
		// If duplicate key error, return empty map - we'll fetch after transaction

		return created, nil
	})
	if err != nil {
		return err
	}

	// Handle any race condition cases by fetching them
	for _, key := range missingKeys {
		if ns, ok := createdNamespaces[key]; ok {
			namespaces[key] = ns
			// Cache by both name and ID
			h.RatelimitNamespaceCache.Set(ctx, key, ns)
			idKey := cache.ScopedKey{WorkspaceID: key.WorkspaceID, Key: ns.ID}
			h.RatelimitNamespaceCache.Set(ctx, idKey, ns)
		} else {
			// Namespace was created by another request, fetch it
			loader, err := namespaceLoader(ctx, []cache.ScopedKey{key})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Public("Failed to fetch namespace after race condition."),
				)
			}

			if ns, ok := loader[key]; ok {
				namespaces[key] = ns
				// Cache by both name and ID
				h.RatelimitNamespaceCache.Set(ctx, key, ns)
				idKey := cache.ScopedKey{WorkspaceID: key.WorkspaceID, Key: ns.ID}
				h.RatelimitNamespaceCache.Set(ctx, idKey, ns)
			} else {
				return fault.New("namespace not found after duplicate key error")
			}
		}
	}

	return nil
}

type checkMeta struct {
	namespaceName string
	namespaceID   string
	identifier    string
	overrideID    string
	limit         int64
}

func getLimitAndDuration(check openapi.V2RatelimitLimitRequestBody, namespace db.FindRatelimitNamespace) (int64, int64, string, error) {
	override, found, err := matchOverride(check.Identifier, namespace)
	if err != nil {
		return 0, 0, "", err
	}

	if found {
		return override.Limit, override.Duration, override.ID, nil
	}

	return check.Limit, check.Duration, "", nil
}

func matchOverride(identifier string, namespace db.FindRatelimitNamespace) (db.FindRatelimitNamespaceLimitOverride, bool, error) {
	if override, ok := namespace.DirectOverrides[identifier]; ok {
		return override, true, nil
	}

	for _, override := range namespace.WildcardOverrides {
		ok, err := match.Wildcard(identifier, override.Identifier)
		if err != nil {
			return db.FindRatelimitNamespaceLimitOverride{}, false, err
		}

		if !ok {
			continue
		}

		return override, true, nil
	}

	return db.FindRatelimitNamespaceLimitOverride{
		Limit:      0,
		ID:         "",
		Identifier: "",
		Duration:   0,
	}, false, nil
}
