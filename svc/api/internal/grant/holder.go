package grant

import (
	"database/sql"
	"strings"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Holder is who holds a session: the Unkey CLI or one registered OAuth client.
// It is comparable, so refresh can compare the presenter with the session's
// holder. The zero value is the CLI, mirroring client_id IS NULL in storage.
type Holder struct {
	clientID ClientID
}

// CLI holds the sessions device login issues.
var CLI = Holder{clientID: ""}

// HeldBy is the holder for a registered client. It rejects an id registration
// could not have issued, so a missing client_id never turns an OAuth request
// into a CLI one.
func HeldBy(id ClientID) (Holder, error) {
	if !strings.HasPrefix(string(id), string(uid.OAuthClientPrefix)+"_") {
		return Holder{}, fault.New("client_id is not an oauth client id",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("client_id is invalid."),
		)
	}
	return Holder{clientID: id}, nil
}

func holderOf(row db.CliSession) Holder {
	if !row.ClientID.Valid {
		return CLI
	}
	return Holder{clientID: ClientID(row.ClientID.String)}
}

func (h Holder) nullClientID() sql.NullString {
	return sql.NullString{String: string(h.clientID), Valid: h.clientID != ""}
}

// sessionAudit is the audit vocabulary for one holder's sessions. CLI sessions
// keep their events; OAuth sessions get their own family so the audit log
// separates them without reading metadata.
type sessionAudit struct {
	create  auditlog.AuditLogEvent
	refresh auditlog.AuditLogEvent
	revoke  auditlog.AuditLogEvent
	target  auditlog.AuditLogTargetType
	display string
}

func (h Holder) audit() sessionAudit {
	if h.clientID == "" {
		return sessionAudit{
			create:  auditlog.CLISessionCreateEvent,
			refresh: auditlog.CLISessionRefreshEvent,
			revoke:  auditlog.CLISessionRevokeEvent,
			target:  auditlog.CLISessionTargetType,
			display: "CLI session",
		}
	}
	return sessionAudit{
		create:  auditlog.OAuthSessionCreateEvent,
		refresh: auditlog.OAuthSessionRefreshEvent,
		revoke:  auditlog.OAuthSessionRevokeEvent,
		target:  auditlog.OAuthSessionTargetType,
		display: "OAuth session",
	}
}

// actorMeta is the audit actor metadata for a request the holder makes with
// session sessionID, matching what auditactor records for its access tokens.
func (h Holder) actorMeta(sessionID string, label string) map[string]any {
	meta := map[string]any{"via": "cli", "session_id": sessionID, "device": label}
	if h.clientID != "" {
		meta["via"] = "oauth"
		meta["client_id"] = string(h.clientID)
	}
	return meta
}

// targetMeta is the audit target metadata for this holder's sessions.
func (h Holder) targetMeta() map[string]any {
	if h.clientID == "" {
		return map[string]any{}
	}
	return map[string]any{"client_id": string(h.clientID)}
}

// noun names this holder's sessions in user-facing messages.
func (h Holder) noun() string {
	if h.clientID == "" {
		return "CLI session"
	}
	return "session"
}

// signInAgain is the next step when this holder's session ends. label is the
// session's device name; for an OAuth client it is the client's name.
func (h Holder) signInAgain(label string) string {
	return clisession.SignInAgain(string(h.clientID), label)
}
