package grant

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Poll returns the login state. The first poll after approval creates the
// session and returns tokens once.
func (s *Service) Poll(ctx context.Context, req PollRequest) (PollResult, error) {
	if err := s.requireEnabled(); err != nil {
		return PollResult{}, err
	}
	if err := s.limit(ctx, "poll", req.RemoteIP, pollLimitPerMinute); err != nil {
		return PollResult{}, err
	}

	row, err := db.Query.FindDeviceAuthorizationByID(ctx, s.DB.RW(), strings.TrimSpace(req.LoginID))
	if err != nil {
		if db.IsNotFound(err) {
			return PollResult{}, cliLoginNotFound()
		}
		return PollResult{}, err
	}
	parsed, err := parseRequest(row)
	if err != nil {
		return PollResult{}, err
	}
	if _, ok := parsed.delivery.(deviceDelivery); !ok {
		return PollResult{}, cliLoginNotFound()
	}

	switch s.authorizationStatus(row) {
	case db.DeviceAuthorizationsStatusDenied:
		return pollResult(PollDenied, "Authorization was denied."), nil
	case db.DeviceAuthorizationsStatusExpired:
		return pollResult(PollExpired, "The login code expired. Run unkey login again."), nil
	case db.DeviceAuthorizationsStatusApproved:
		if row.ApproverUserID.Valid && row.WorkspaceID.Valid {
			return s.pollIssue(ctx, parsed, req)
		}
	case db.DeviceAuthorizationsStatusConsumed:
		return PollResult{}, alreadyIssued()
	case db.DeviceAuthorizationsStatusPending:
	}
	return pollResult(PollPermissionsRequired, ""), nil
}

func (s *Service) pollIssue(ctx context.Context, parsed request, req PollRequest) (PollResult, error) {
	res, err := s.issue(ctx, parsed, req.RemoteIP, req.UserAgent)
	if err != nil {
		return PollResult{}, err
	}
	switch res.outcome {
	case issueComplete:
		return PollResult{
			Status:   PollComplete,
			Interval: pollIntervalSeconds,
			Detail:   "",
			Tokens:   res.tokens,
		}, nil
	case issueDenied:
		return pollResult(PollDenied, res.detail), nil
	case issueChanged:
		return pollResult(PollPermissionsRequired, ""), nil
	case issueReplayed:
		return PollResult{}, alreadyIssued()
	default:
		return PollResult{}, fault.New("unexpected issue outcome")
	}
}

func pollResult(status PollStatus, detail string) PollResult {
	return PollResult{
		Status:   status,
		Interval: pollIntervalSeconds,
		Detail:   detail,
		Tokens:   Tokens{AccessToken: "", RefreshToken: "", ExpiresIn: 0, WorkspaceID: "", SessionID: ""},
	}
}

// issueOutcome is what issue did with an approved request.
type issueOutcome int

const (
	// issueComplete created the session; tokens are set.
	issueComplete issueOutcome = iota
	// issueDenied found the workspace unusable or the narrowed permissions
	// empty, and denied the request. detail says which.
	issueDenied
	// issueChanged found the request no longer approved as read: it was
	// re-approved, denied, or expired between the read and the lock.
	issueChanged
	// issueReplayed found the request already consumed. sessionID is the
	// session it issued.
	issueReplayed
)

type issueResult struct {
	outcome   issueOutcome
	detail    string
	sessionID string
	tokens    Tokens
}

// issue turns an approved request into a session held by the request's
// holder. Poll and Redeem both end here. WorkOS and permission narrowing run
// before the transaction; the transaction locks the row, re-checks that it is
// still approved exactly as read (including the authorization code hash), and
// consumes it. Errors are infrastructure failures; every refusal is an
// outcome the caller words for its protocol.
func (s *Service) issue(ctx context.Context, req request, remoteIP string, userAgent string) (issueResult, error) {
	row := req.row
	holder := req.delivery.holder()
	workspace, err := db.Query.FindWorkspaceByID(ctx, s.DB.RO(), row.WorkspaceID.String)
	if err != nil {
		return issueResult{}, err
	}
	if !workspaceUsable(workspace) {
		return s.denyIssue(ctx, row.ID, "This workspace cannot approve a "+holder.loginNoun()+".")
	}

	roles, found, err := s.Memberships.ActiveRoles(ctx, workspace.OrgID, row.ApproverUserID.String)
	if err != nil {
		return issueResult{}, err
	}
	requested, err := decodePermissions(row.Permissions)
	if err != nil {
		return issueResult{}, err
	}
	var narrowed []string
	if found {
		narrowed, err = intersectPermissions(ctx, workspace.ID, roles, requested)
		if err != nil {
			return issueResult{}, err
		}
	}
	if len(narrowed) == 0 {
		return s.denyIssue(ctx, row.ID, "You no longer have permission to finish this login.")
	}

	sessionID := uid.New(uid.CLISessionPrefix)
	family := newRefreshFamily()
	refreshToken := newRefreshToken(family)
	now := s.Clock.Now().UnixMilli()
	permsJSON, err := json.Marshal(narrowed)
	if err != nil {
		return issueResult{}, err
	}
	accessToken, err := s.signAccessToken(row.ApproverUserID.String, row.ApproverName.String, workspace.ID, sessionID, row.DeviceName.String, narrowed)
	if err != nil {
		return issueResult{}, err
	}

	var result issueResult
	err = db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		locked, err := db.Query.FindDeviceAuthorizationByIDForUpdate(ctx, tx, row.ID)
		if err != nil {
			return err
		}
		switch s.authorizationStatus(locked) {
		case db.DeviceAuthorizationsStatusConsumed:
			result = refused(issueReplayed, "", locked.SessionID.String)
			return nil
		case db.DeviceAuthorizationsStatusApproved:
			if approvalChanged(row, locked) {
				result = refused(issueChanged, "", "")
				return nil
			}
		case db.DeviceAuthorizationsStatusPending, db.DeviceAuthorizationsStatusDenied, db.DeviceAuthorizationsStatusExpired:
			result = refused(issueChanged, "", "")
			return nil
		}

		if err := db.Query.InsertCLISession(ctx, tx, db.InsertCLISessionParams{
			ID:               sessionID,
			WorkspaceID:      workspace.ID,
			UserID:           locked.ApproverUserID.String,
			UserName:         locked.ApproverName,
			Permissions:      permsJSON,
			DeviceName:       locked.DeviceName,
			RefreshTokenHash: hashRefreshToken(refreshToken),
			RefreshFamily:    family,
			ClientID:         holder.nullClientID(),
			CreatedAt:        now,
			LastUsedAt:       now,
			ExpiresAt:        sessionExpiry(now, now),
		}); err != nil {
			return err
		}
		n, err := db.Query.ConsumeDeviceAuthorization(ctx, tx, db.ConsumeDeviceAuthorizationParams{
			SessionID: sql.NullString{
				String: sessionID,
				Valid:  true,
			},
			ID: locked.ID,
		})
		if err != nil {
			return err
		}
		if err := assert.Equal(n, int64(1), "a locked approved request must consume exactly once"); err != nil {
			return err
		}

		audit := holder.audit()
		ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
		if err := s.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			Event:         audit.create,
			Display:       "Created " + audit.display + " " + sessionID,
			WorkspaceID:   workspace.ID,
			ActorType:     auditlog.UserActor,
			ActorID:       locked.ApproverUserID.String,
			ActorName:     locked.ApproverName.String,
			ActorMeta:     map[string]any{},
			CorrelationID: "",
			RemoteIP:      remoteIP,
			UserAgent:     userAgent,
			Targets: []auditlog.AuditLogTarget{{
				Type:        audit.target,
				ID:          sessionID,
				Name:        locked.DeviceName.String,
				DisplayName: locked.DeviceName.String,
				Meta:        holder.targetMeta(),
			}},
		}}); err != nil {
			return err
		}

		result = issueResult{
			outcome:   issueComplete,
			detail:    "",
			sessionID: sessionID,
			tokens: Tokens{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresIn:    int64(accessTTL / time.Second),
				WorkspaceID:  workspace.ID,
				SessionID:    sessionID,
			},
		}
		return nil
	})
	return result, err
}

// approvalChanged reports whether the locked row differs from the approval
// issue read. Re-approval changes approved_at and, for an OAuth request, mints
// a new code hash, so a code from an earlier approval fails here.
func approvalChanged(read db.DeviceAuthorization, locked db.DeviceAuthorization) bool {
	return read.ApprovedAt != locked.ApprovedAt ||
		read.ApproverUserID != locked.ApproverUserID ||
		read.WorkspaceID != locked.WorkspaceID ||
		read.CodeHash != locked.CodeHash ||
		!bytes.Equal(read.Permissions, locked.Permissions)
}

func (s *Service) denyIssue(ctx context.Context, id string, detail string) (issueResult, error) {
	if err := s.denyAuthorization(ctx, id); err != nil {
		return issueResult{}, err
	}
	return refused(issueDenied, detail, ""), nil
}

func refused(outcome issueOutcome, detail string, sessionID string) issueResult {
	var tokens Tokens
	return issueResult{outcome: outcome, detail: detail, sessionID: sessionID, tokens: tokens}
}

func (s *Service) denyAuthorization(ctx context.Context, id string) error {
	n, err := db.Query.DenyDeviceAuthorization(ctx, s.DB.RW(), id)
	if err != nil {
		return err
	}
	if n != 1 {
		return alreadyIssued()
	}
	return nil
}
