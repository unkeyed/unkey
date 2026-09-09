package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	// Aliased: this handler's local `portal` variable is the db.Portal row it
	// resolves, which would otherwise shadow the package.
	portalservice "github.com/unkeyed/unkey/internal/services/portal"
	"github.com/unkeyed/unkey/pkg/auditlog"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/internal/policyconfig"
	portalrules "github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/openapi"
	// The key requirements below are owned by the operator routes that
	// enforce them, so this route borrows them rather than restating them.
	listkeys "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
	rerollkey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_reroll_key"
)

type (
	Request  = openapi.V2PortalCreateSessionRequestBody
	Response = openapi.V2PortalCreateSessionResponseBody
)

// Handler implements zen.Route for the portal session creation endpoint.
type Handler struct {
	DB            db.Database
	Auditlogs     auditlogs.AuditLogService
	PortalBaseURL string
	Clock         clock.Clock
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.createSession" }

// resolveKeyspaceIDs derives the keyspaces a portal session is scoped to from
// the portal. A portal maps to exactly one of a keyspace or an app:
//
//   - keyspace-mapped (key_auth_id): the configured keyspace scopes key
//     capabilities directly.
//   - app-mapped (app_id): the app's current deployment carries a gateway
//     policy config whose keyauth policies list the keyspaces it verifies keys
//     against at the gateway; those keySpaceIds become the session's keyspaces.
//
// The portal is bound to the caller's workspace, so the resolved keyspaces can
// never belong to another workspace.
func (h *Handler) resolveKeyspaceIDs(ctx context.Context, workspaceID string, portal db.Portal) ([]string, error) {
	hasKeyspace := portal.KeyAuthID.Valid
	hasApp := portal.AppID.Valid

	// A well-formed portal maps to exactly one of a keyspace or an app. Neither
	// or both is a misconfiguration the session can't be scoped from.
	if hasKeyspace == hasApp {
		return nil, fault.New("portal not mapped to exactly one target",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("portal must reference exactly one of key_auth_id or app_id"),
			fault.Public("Portal configuration is invalid."),
		)
	}

	if hasKeyspace {
		return []string{portal.KeyAuthID.String}, nil
	}

	raw, err := db.Query.FindAppPolicyConfigByID(ctx, h.DB.RO(), db.FindAppPolicyConfigByIDParams{
		AppID:       portal.AppID.String,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fault.New("portal app has no current deployment",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Internal("app has no current deployment to resolve keyspaces from"),
				fault.Public("Portal is not available: the app has no active deployment."),
			)
		}
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up app gateway policy config"),
			fault.Public("Failed to look up portal configuration."),
		)
	}

	keyspaceIDs, err := keyspacesFromPolicyConfig(raw)
	if err != nil {
		return nil, err
	}
	if len(keyspaceIDs) == 0 {
		return nil, fault.New("portal app has no keyauth policies",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal("app gateway policy config declares no keyauth keyspaces"),
			fault.Public("Portal is not available: the app has no key verification configured."),
		)
	}

	return keyspaceIDs, nil
}

// keyspacesFromPolicyConfig parses a deployment's gateway policy config and returns
// the deduplicated keyspaces declared across its keyauth policies. Empty or
// legacy empty-object configs yield no keyspaces.
func keyspacesFromPolicyConfig(raw []byte) ([]string, error) {
	cfg, err := policyconfig.Parse(raw)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to unmarshal app gateway policy config"),
			fault.Public("Portal configuration is invalid."),
		)
	}

	seen := make(map[string]struct{})
	var keyspaceIDs []string
	for _, p := range cfg.GetPolicies() {
		for _, ks := range p.GetKeyauth().GetKeySpaceIds() {
			if _, ok := seen[ks]; ok {
				continue
			}
			seen[ks] = struct{}{}
			keyspaceIDs = append(keyspaceIDs, ks)
		}
	}
	return keyspaceIDs, nil
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Request shape only, so it runs before the portal lookup.
	if err = validateScopeCombination(req.Scopes); err != nil {
		return err
	}

	// Also enforced at the mint seam. Running it before the lookup keeps a
	// refused credential from learning which portals exist, since past this
	// point an absent portal answers 404 and a present one 403.
	if err = requireRootKeyCredential(principal); err != nil {
		return err
	}

	workspaceID := principal.AuthorizedWorkspaceID

	portal, err := db.Query.FindPortalByIdOrSlug(ctx, h.DB.RO(), db.FindPortalByIdOrSlugParams{
		WorkspaceID: workspaceID,
		Portal:      req.Portal,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("portal not found",
				fault.Code(codes.Data.Portal.NotFound.URN()),
				fault.Internal("no portal found for the given id or slug"),
				fault.Public("Portal not found."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal"),
			fault.Public("Failed to look up portal configuration."),
		)
	}

	// Stage 1: may this caller mint a session for *this* portal at all?
	//
	// It runs before the enabled check so the tuple can name a concrete portal,
	// and it is built from portal.ID rather than req.Portal: req.Portal accepts
	// an id or a slug, legacy tuples match literally, and a slug-shaped tuple
	// could never match a dashboard-granted portal.pc_*.create_portal_session.
	// The wildcard tuple branch is spelled out for the same reason: `*` in a
	// stored legacy grant is matched literally, it does not expand.
	//
	// No arm names the portal resource itself: administering a portal must not
	// imply minting sessions for its end users. A stored `**#*` still satisfies
	// the canonical arm, so requireRootKeyCredential above is what actually
	// keeps a dashboard admin from minting.
	if err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Portal,
			ResourceID:   "*",
			Action:       rbac.CreatePortalSession,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Portal,
			ResourceID:   portal.ID,
			Action:       rbac.CreatePortalSession,
		}),
		rbac.U(
			urn.New().Workspace(workspaceID).Project(portal.ProjectID).Portal(portal.ID).Session("*"),
			permissions.Write,
		),
	)); err != nil {
		// Masked as 404 so a caller short of the minting permission cannot tell an
		// existing portal from an absent one, or learn the resolved portal id --
		// req.Portal accepts a slug, so that id is otherwise unobtainable. The
		// helper builds a fresh chain rather than wrapping, which matters here:
		// UserFacingMessage concatenates every public message, so a wrap would
		// append the rendered query naming that id. Principal.Authorize already
		// logs the denied query and the granted set. Anything that is not an
		// authorization denial passes through unmasked.
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Portal.NotFound.URN(),
			"Portal not found.",
		)
	}

	if !portal.Enabled {
		return fault.New("portal is disabled",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal("portal is disabled"),
			fault.Public("Portal is disabled."),
		)
	}

	// The keyspaces a session is scoped to come from the portal, not the public
	// request: the portal is already bound to this workspace, so key
	// capabilities can never reach another workspace's keyspaces.
	keyspaceIDs, err := h.resolveKeyspaceIDs(ctx, workspaceID, portal)
	if err != nil {
		return err
	}

	// Stage 2: a minted session may never carry a capability the calling root
	// key does not itself hold. This precedes the exchange code, the session
	// insert and the audit log, so a rejection writes nothing.
	if err = h.authorizeScopes(ctx, principal, portal, keyspaceIDs, req.Scopes); err != nil {
		return err
	}

	now := h.Clock.Now()

	verbs := make([]string, len(req.Scopes))
	for i, p := range req.Scopes {
		verbs[i] = string(p)
	}

	scopesJSON, err := json.Marshal(portalservice.Grant{
		KeyspaceIDs: keyspaceIDs,
		Scopes:      verbs,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to marshal portal session grant"),
			fault.Public("An internal error occurred."),
		)
	}

	preview := false
	if req.Preview != nil {
		preview = *req.Preview
	}

	// Optional, and an empty string is treated as absent: a portal with no
	// return URL simply shows no return link.
	returnURL := sql.NullString{Valid: false, String: ""}
	if req.ReturnUrl != nil && *req.ReturnUrl != "" {
		// Validated here rather than trusted from the spec. The field carries
		// `format: uri`, which accepts `javascript:...` and is not asserted by the
		// request validator anyway, and the portal renders this value as an anchor
		// href -- so an unchecked scheme executes in the end user's browser, with
		// the portal's origin.
		if err = portalrules.ValidateReturnURL(*req.ReturnUrl); err != nil {
			return err
		}
		returnURL = sql.NullString{Valid: true, String: *req.ReturnUrl}
	}

	sessionID, exchangeCode, err := h.mintSession(ctx, s, principal, mintRequest{
		Portal:      portal,
		ExternalID:  req.ExternalId,
		Scopes:      verbs,
		KeyspaceIDs: keyspaceIDs,
		ScopesJSON:  scopesJSON,
		Preview:     preview,
		ReturnURL:   returnURL,
		Now:         now,
	})
	if err != nil {
		return err
	}

	portalURL := fmt.Sprintf("%s/?code=%s", h.PortalBaseURL, exchangeCode)

	s.ResponseWriter().Header().Set("Cache-Control", "no-store")
	s.ResponseWriter().Header().Set("Pragma", "no-cache")

	// The URL already carries the exchange code, so the response returns the
	// non-secret session handle rather than duplicating the credential.
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2PortalCreateSessionResponseData{
			Id:  sessionID,
			Url: portalURL,
		},
	})
}

// mintRequest carries one portal session's row and audit fields, already
// resolved and authorized by the caller.
type mintRequest struct {
	Portal      db.Portal
	ExternalID  string
	Scopes      []string
	KeyspaceIDs []string
	ScopesJSON  []byte
	Preview     bool
	ReturnURL   sql.NullString
	Now         time.Time
}

// mintSession is the single seam every portal session is minted through, so
// [requireRootKeyCredential] restricts the capability rather than one route.
func (h *Handler) mintSession(
	ctx context.Context,
	s *zen.Session,
	principal *authprincipal.Principal,
	req mintRequest,
) (string, string, error) {
	if err := requireRootKeyCredential(principal); err != nil {
		return "", "", err
	}

	workspaceID := principal.AuthorizedWorkspaceID
	sessionID := uid.New(uid.PortalSessionPrefix)

	// The exchange code is a bearer credential: it is returned to the caller
	// once, embedded in the redirect URL, and stored only as a hash.
	exchangeCode := string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()
	exchangeCodeExpiresAt := req.Now.Add(15 * time.Minute).UnixMilli()

	err := db.Tx(ctx, h.DB.RW(), func(txCtx context.Context, tx db.DBTX) error {
		// Re-read on the primary inside the write transaction. The resolve above
		// runs on the read-only connection, so a portal deleted moments earlier can
		// still appear live there.
		//
		// This matters because deleting a portal revokes its sessions: revocation
		// only touches rows that exist when it runs, so a session minted in the
		// replica-lag window would survive the delete, and once the portal row is
		// gone nothing can revoke it afterwards. Losing the race here costs the
		// caller a retry; losing it silently costs an end user access that was
		// supposed to be cut.
		if _, txErr := db.Query.FindPortalByIdOrSlug(txCtx, tx, db.FindPortalByIdOrSlugParams{
			WorkspaceID: workspaceID,
			Portal:      req.Portal.ID,
		}); txErr != nil {
			if db.IsNotFound(txErr) {
				return fault.New("portal not found",
					fault.Code(codes.Data.Portal.NotFound.URN()),
					fault.Internal(fmt.Sprintf("portal %s was deleted between the replica read and the session insert", req.Portal.ID)),
					fault.Public("Portal not found."),
				)
			}
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error re-reading portal before minting a session"),
				fault.Public("Failed to create session."),
			)
		}

		if txErr := db.Query.InsertPortalSession(txCtx, tx, db.InsertPortalSessionParams{
			ID:                    sessionID,
			WorkspaceID:           workspaceID,
			PortalID:              req.Portal.ID,
			ExternalID:            req.ExternalID,
			Scopes:                req.ScopesJSON,
			Preview:               req.Preview,
			ExchangeCodeHash:      hash.Sha256(exchangeCode),
			ExchangeCodeExpiresAt: exchangeCodeExpiresAt,
			ReturnUrl:             req.ReturnURL,
			CreatedAt:             req.Now.UnixMilli(),
		}); txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to insert portal session"),
				fault.Public("Failed to create session."),
			)
		}

		// A reroll performed later inside the session is attributed to the end
		// user, so this entry is the only place the root key and the end user
		// it acts for are joined.
		if txErr := h.Auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
			{
				Event:         auditlog.PortalSessionCreateEvent,
				WorkspaceID:   workspaceID,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Created portal session for %s", req.ExternalID),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          sessionID,
						DisplayName: req.ExternalID,
						Name:        req.ExternalID,
						Meta: map[string]any{
							"portalId":    req.Portal.ID,
							"slug":        req.Portal.Slug,
							"scopes":      req.Scopes,
							"keyspaceIds": req.KeyspaceIDs,
						},
						Type: auditlog.PortalSessionResourceType,
					},
				},
			},
		}); txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to insert audit log"),
				fault.Public("Failed to create session."),
			)
		}

		return nil
	})
	if err != nil {
		return "", "", err
	}

	return sessionID, exchangeCode, nil
}

// requireRootKeyCredential admits only a root key to mint a portal session.
//
// This is the only control standing between a dashboard token and minting: the
// WorkOS admin role grants unkey:v1:{workspace}:**#*, which covers the portal
// session URN stage 1 evaluates, so without this any workspace admin could mint
// a session authenticating as an arbitrary end user. Naming every enum member
// keeps a future credential type failing the exhaustive linter until someone
// decides whether it may mint.
func requireRootKeyCredential(principal *authprincipal.Principal) error {
	var mayMint bool
	switch principal.Type {
	case authprincipal.TypeAPIKey:
		// Only root keys carry this type in svc/api: the resolver that produces
		// it rejects any key that is not a root key.
		mayMint = true
	case authprincipal.TypeJWT, authprincipal.TypePortalSession:
		mayMint = false
	}

	if mayMint {
		return nil
	}

	return fault.New("portal session minting requires a root key",
		fault.Code(codes.Auth.Authorization.Forbidden.URN()),
		fault.Internal(fmt.Sprintf("credential type %q may not mint portal sessions", principal.Type)),
		fault.Public("Portal sessions can only be created with a root key."),
	)
}

// validateScopeCombination rejects a scope set the portal cannot serve.
//
// The portal reaches rerolling from the keys page, so a reroll-only session
// mints fine and then strands the end user with no page to open. The enum
// constrains each item, not the combination, so the check lives here.
func validateScopeCombination(scopes []openapi.V2PortalCreateSessionRequestBodyScopes) error {
	var hasRead, hasReroll bool
	for _, scope := range scopes {
		switch scope {
		case openapi.KeysRead:
			hasRead = true
		case openapi.KeysReroll:
			hasReroll = true
		}
	}

	if hasReroll && !hasRead {
		return fault.New("keys:reroll requires keys:read",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("scopes contained keys:reroll without keys:read"),
			fault.Public("The \"keys:reroll\" scope requires \"keys:read\" in the same session."),
		)
	}

	return nil
}

// ScopeQueries returns the authorization requirements the *calling* root key
// must satisfy for one requested portal scope on one keyspace.
//
// The mapping is a total function over the scope enum, and the ok result is what
// makes that checkable: rbac.And over zero children evaluates to valid, so a
// scope that were silently skipped would mint a session with no check at all.
// An unrecognized scope therefore reports ok=false and the caller denies.
//
// It is exported so the deny-by-default behaviour can be tested directly. The
// OpenAPI enum rejects unknown values at the request boundary, so there is no
// way to reach the default arm through the route itself.
func ScopeQueries(
	scope openapi.V2PortalCreateSessionRequestBodyScopes,
	apiID string,
	storeEncryptedKeys bool,
) ([]rbac.PermissionQuery, bool) {
	switch scope {
	case openapi.KeysRead:
		return []rbac.PermissionQuery{listkeys.ReadKeysPermissions(apiID)}, true

	case openapi.KeysReroll:
		// Rerolling is a create, matching what the operator reroll route
		// requires. The encryption conjunct is keyspace-conditional: a portal
		// session that can reroll a key in a keyspace storing recoverable key
		// material also hands out that material.
		//
		// The conjunct keys off the keyspace flag rather than an individual
		// key's encryption row because mint time cannot know which key a
		// session will later reroll. That makes it a conservative proxy
		// that can go stale, which is safe today: the reroll core gates both
		// the encryption operation and the legacy encrypt_key check on the key
		// itself, so turning a keyspace's encryption on does not
		// make already-existing keys recoverable and gives a live session
		// nothing new.
		//
		// One path would escalate once UpdateKeySpaceKeyEncryption gains a
		// production caller: a keyspace toggled on, off, then on again around a
		// mint. That belongs to the toggle, which must invalidate live portal
		// sessions on a keyspace when it turns encryption on. Do not close it
		// here by requiring encrypt_key unconditionally: that would make this
		// ceiling stricter than the operator route it exists to mirror.
		queries := []rbac.PermissionQuery{rerollkey.CreateKeyPermissions(apiID)}
		if storeEncryptedKeys {
			queries = append(queries, rerollkey.EncryptKeyPermissions(apiID))
		}
		return queries, true

	default:
		return nil, false
	}
}

// CanonicalScopeQueries returns the canonical form of the requirement
// [ScopeQueries] expresses in legacy tuples, for one scope on one keyspace.
//
// The ok flag is load-bearing: rbac.And over zero children evaluates to valid,
// so an unmapped scope must be dropped from the composition rather than
// contribute an empty conjunction.
//
// Unlike the legacy arms, these are hand-written rather than borrowed from the
// operator routes they mirror (v2_apis_list_keys for read, v2_keys_reroll_key
// for reroll), so they can drift weaker than those routes without failing.
func CanonicalScopeQueries(
	scope openapi.V2PortalCreateSessionRequestBodyScopes,
	workspaceID string,
	projectID string,
	keyspaceID string,
	storeEncryptedKeys bool,
) ([]rbac.PermissionQuery, bool) {
	keyspace := urn.New().Workspace(workspaceID).Project(projectID).Keyspace(keyspaceID)

	// Mint time has no key id, so key-scoped requirements are wildcards: a
	// caller holding a grant on one concrete key cannot mint.
	anyKey := keyspace.Key(permissions.Wildcard)

	switch scope {
	case openapi.KeysRead:
		return []rbac.PermissionQuery{
			rbac.U(anyKey, permissions.Read),
			rbac.U(keyspace, permissions.Read),
		}, true

	case openapi.KeysReroll:
		// The reroll route's canonical create and encryption arms are the same
		// key-write leaf, so the encryption conjunct is degenerate here. It is
		// spelled out anyway to mirror the route: a ceiling stricter than the
		// route would refuse mints the operator endpoint allows.
		queries := []rbac.PermissionQuery{rbac.U(anyKey, permissions.Write)}
		if storeEncryptedKeys {
			queries = append(queries, rbac.U(anyKey, permissions.Write))
		}
		return queries, true

	default:
		return nil, false
	}
}

// authorizeScopes enforces the mint-time ceiling: for every requested scope, the
// caller must hold the equivalent operator permission on every keyspace the
// session will be scoped to. A caller short of any one of them is refused
// outright rather than handed the intersection, so a missing grant surfaces as a
// 403 instead of a silently degraded portal.
func (h *Handler) authorizeScopes(
	ctx context.Context,
	principal *authprincipal.Principal,
	portal db.Portal,
	keyspaceIDs []string,
	scopes []openapi.V2PortalCreateSessionRequestBodyScopes,
) error {
	workspaceID := portal.WorkspaceID

	// Fail closed. Every check below is a conjunction, and rbac.And over an
	// empty child list is valid, so an empty keyspace or scope list would mint
	// an unchecked session.
	if len(keyspaceIDs) == 0 || len(scopes) == 0 {
		return fault.New("nothing to authorize the portal session against",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("portal session must resolve at least one keyspace and one scope"),
			fault.Public("Portal configuration is invalid."),
		)
	}

	owners, err := h.ownersByKeyspace(ctx, workspaceID, keyspaceIDs)
	if err != nil {
		return err
	}

	if err = requireSameProject(portal, keyspaceIDs, owners); err != nil {
		return err
	}

	encrypted, err := h.encryptionByKeyspace(ctx, workspaceID, keyspaceIDs)
	if err != nil {
		return err
	}

	for _, scope := range scopes {
		var checks []rbac.PermissionQuery

		for _, keyspaceID := range keyspaceIDs {
			owner := owners[keyspaceID]

			legacy, ok := ScopeQueries(scope, owner.APIID, encrypted[keyspaceID])
			if !ok {
				// Reaching this means the request enum and the mapping have
				// diverged, which is a server bug rather than a caller problem.
				// Same defect class as the empty-checks guard below, so same code.
				return fault.New("unmapped portal scope",
					fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Internal(fmt.Sprintf("scope %q has no authorization mapping", scope)),
					fault.Public("Portal configuration is invalid."),
				)
			}

			// One disjunction per keyspace rather than a flat conjunction over
			// all of them: flattening would let a caller satisfy one keyspace
			// in legacy tuples and another canonically.
			arms := []rbac.PermissionQuery{rbac.And(legacy...)}
			if canonical, hasCanonical := CanonicalScopeQueries(
				scope, workspaceID, owner.ProjectID, keyspaceID, encrypted[keyspaceID],
			); hasCanonical {
				arms = append(arms, rbac.And(canonical...))
			}

			checks = append(checks, rbac.Or(arms...))
		}

		if len(checks) == 0 {
			return fault.New("no authorization checks for portal scope",
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal(fmt.Sprintf("scope %q produced no authorization checks", scope)),
				fault.Public("Portal configuration is invalid."),
			)
		}

		if err = principal.Authorize(rbac.And(checks...)); err != nil {
			// Fresh chain for the same reason as stage 1: the rendered query names
			// the api ids behind this portal's keyspaces, which is more than the
			// caller needs to know they are short a grant.
			return fault.New("insufficient permissions for requested scope",
				fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.Internal(fmt.Sprintf("stage 2 denied for scope %q: %s", scope, fault.InternalMessage(err))),
				fault.Public(fmt.Sprintf("You do not have permission to grant the %q scope to a portal session.", scope)),
			)
		}
	}

	return nil
}

// requireSameProject refuses a portal whose resolved keyspaces reach outside
// its own project, so the portal's stored project can be trusted as the project
// segment of the canonical requirements above. Otherwise a deployment naming
// another project's keyspace would be checked, and granted, under the wrong
// project.
//
// Ids stay out of the public message because this runs before the per-keyspace
// check, so the caller has proven no grant on the keyspace it names.
func requireSameProject(portal db.Portal, keyspaceIDs []string, owners map[string]keyspaceOwner) error {
	for _, keyspaceID := range keyspaceIDs {
		owner := owners[keyspaceID]
		if owner.ProjectID == portal.ProjectID {
			continue
		}

		logger.Error("portal resolved a keyspace outside its project",
			slog.String("workspace_id", portal.WorkspaceID),
			slog.String("portal_id", portal.ID),
			slog.String("portal_project_id", portal.ProjectID),
			slog.String("keyspace_id", keyspaceID),
			slog.String("keyspace_project_id", owner.ProjectID),
		)

		return fault.New("portal keyspace belongs to another project",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal(fmt.Sprintf(
				"portal %s in project %s resolved keyspace %s owned by project %s",
				portal.ID, portal.ProjectID, keyspaceID, owner.ProjectID,
			)),
			fault.Public("Portal is not available: it serves a keyspace outside its project. Fix the app's deployment configuration."),
		)
	}

	return nil
}

// keyspaceOwner is the api and project that own one resolved keyspace.
type keyspaceOwner struct {
	APIID     string
	ProjectID string
}

// ownersByKeyspace maps each resolved keyspace to the api and project that own
// it.
//
// Every legacy stage-2 requirement is api-scoped, so a keyspace with no api
// admits no expressible check. That is a misconfiguration rather than a caller
// error: skipping such a keyspace would leave it unchecked, so it fails loudly
// and names the keyspace.
func (h *Handler) ownersByKeyspace(ctx context.Context, workspaceID string, keyspaceIDs []string) (map[string]keyspaceOwner, error) {
	rows, err := db.Query.FindApisByKeyAuthIds(ctx, h.DB.RO(), db.FindApisByKeyAuthIdsParams{
		WorkspaceID: workspaceID,
		KeyAuthIds:  keyspaceIDs,
	})
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error resolving apis for portal keyspaces"),
			fault.Public("Failed to look up portal configuration."),
		)
	}

	owners := make(map[string]keyspaceOwner, len(rows))
	for _, row := range rows {
		owners[row.KeyAuthID] = keyspaceOwner{APIID: row.ApiID, ProjectID: row.ProjectID}
	}

	for _, keyspaceID := range keyspaceIDs {
		if owners[keyspaceID].APIID == "" {
			// Reachable without any misconfiguration on our side: apis.deleteApi
			// soft-deletes the api row and leaves key_auth live, so a customer
			// deleting their own api orphans the keyspace this portal resolves to.
			// Every legacy stage-2 requirement is api-scoped, so the mint cannot be
			// authorized -- but that is the portal being unavailable, not an
			// internal fault, so it mirrors the no-active-deployment branch above.
			return nil, fault.New("portal keyspace has no api",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Internal(fmt.Sprintf("keyspace %s has no live associated api, portal session cannot be authorized", keyspaceID)),
				fault.Public("Portal is not available: the API it uses no longer exists."),
			)
		}
	}

	return owners, nil
}

// encryptionByKeyspace reports, per resolved keyspace, whether it stores
// recoverable key material. A keyspace missing from the result is the same
// misconfiguration apiIDsByKeyspace rejects, so it fails rather than defaulting
// to the weaker no-encryption requirement.
func (h *Handler) encryptionByKeyspace(ctx context.Context, workspaceID string, keyspaceIDs []string) (map[string]bool, error) {
	rows, err := db.Query.FindKeyAuthsByIdsAndWorkspace(ctx, h.DB.RO(), db.FindKeyAuthsByIdsAndWorkspaceParams{
		WorkspaceID: workspaceID,
		KeyAuthIds:  keyspaceIDs,
	})
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error resolving portal keyspaces"),
			fault.Public("Failed to look up portal configuration."),
		)
	}

	encrypted := make(map[string]bool, len(rows))
	found := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		encrypted[row.ID] = row.StoreEncryptedKeys
		found[row.ID] = struct{}{}
	}

	for _, keyspaceID := range keyspaceIDs {
		if _, ok := found[keyspaceID]; !ok {
			return nil, fault.New("portal keyspace not found",
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal(fmt.Sprintf("keyspace %s does not exist in this workspace", keyspaceID)),
				fault.Public("Portal configuration is invalid."),
			)
		}
	}

	return encrypted, nil
}
