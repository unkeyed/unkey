package grant

import (
	"crypto/subtle"
	"database/sql"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	authorizationCodePrefix = "ukc_"
	authorizationCodeLength = 32
)

// delivery is how an approved request reaches its holder. It is sealed: only
// deviceDelivery and redirectDelivery implement it, and only parseRequest
// picks one. Every behaviour that differs by mode is a method here, so a third
// mode does not compile until each behaviour is decided for it.
//
// Entry points that serve one mode assert it (Poll: deviceDelivery, Redeem:
// redirectDelivery) and treat the other mode as not found. That is how a login
// id presented as a code, or a code presented as a login id, is rejected.
type delivery interface {
	// holder is who holds the session this request issues.
	holder() Holder

	// approve is what approving the request at now hands out.
	approve(expiresAt int64, now int64, issuer string) (approval, error)

	// deny is where the approver's browser goes after denying the request.
	deny(issuer string) (Decided, error)

	// consentClient is the client shown on the consent page. label is the
	// row's device_name. Nil for device delivery.
	consentClient(label string) *ConsentClient
}

// approval is what Approve stores and returns for one approval.
type approval struct {
	expiresAt int64
	codeHash  sql.NullString
	decided   Decided
}

// deviceDelivery is the Unkey CLI: it polls with the request id.
type deviceDelivery struct{}

func (deviceDelivery) holder() Holder { return CLI }

func (deviceDelivery) approve(expiresAt int64, _ int64, _ string) (approval, error) {
	return approval{
		expiresAt: expiresAt,
		codeHash:  sql.NullString{},
		decided:   Decided{RedirectTo: ""},
	}, nil
}

func (deviceDelivery) deny(string) (Decided, error) {
	return Decided{RedirectTo: ""}, nil
}

func (deviceDelivery) consentClient(string) *ConsentClient { return nil }

// redirectDelivery is a registered OAuth client reached through the
// approver's browser. Every field was verified when the request was created.
type redirectDelivery struct {
	clientID    ClientID
	redirectURI RedirectURI
	state       string
	challenge   Challenge
}

func (d redirectDelivery) holder() Holder { return Holder{clientID: d.clientID} }

// approve mints a fresh authorization code. Re-approval mints another, so a
// code handed out by an earlier approval stops working.
func (d redirectDelivery) approve(expiresAt int64, now int64, issuer string) (approval, error) {
	if err := assert.NotEmpty(issuer, "an oauth request needs an issuer to approve"); err != nil {
		return approval{}, err
	}
	code := authorizationCodePrefix + uid.Secure(authorizationCodeLength)
	redirectTo, err := d.redirectURI.withCode(code, d.state, issuer)
	if err != nil {
		return approval{}, err
	}
	return approval{
		expiresAt: min(expiresAt, now+codeTTL.Milliseconds()),
		codeHash:  sql.NullString{String: hashCode(code), Valid: true},
		decided:   Decided{RedirectTo: redirectTo},
	}, nil
}

func (d redirectDelivery) deny(issuer string) (Decided, error) {
	if err := assert.NotEmpty(issuer, "an oauth request needs an issuer to deny"); err != nil {
		return Decided{}, err
	}
	redirectTo, err := d.redirectURI.WithError("access_denied", "", d.state, issuer)
	if err != nil {
		return Decided{}, err
	}
	return Decided{RedirectTo: redirectTo}, nil
}

func (d redirectDelivery) consentClient(label string) *ConsentClient {
	return &ConsentClient{
		ID:          d.clientID,
		Name:        label,
		RedirectURI: d.redirectURI.raw,
	}
}

// redeemable checks what the token request presents against what the
// authorization request recorded. A failure says nothing about which check
// failed.
func (d redirectDelivery) redeemable(clientID ClientID, redirectURI string, verifier string) bool {
	clientMatches := subtle.ConstantTimeCompare([]byte(clientID), []byte(d.clientID)) == 1
	redirectMatches := subtle.ConstantTimeCompare([]byte(redirectURI), []byte(d.redirectURI.raw)) == 1
	if !d.redirectURI.presented && redirectURI == "" {
		redirectMatches = true
	}
	verified := d.challenge.Verify(verifier)
	return clientMatches && redirectMatches && verified
}

// request is a parsed device_authorizations row. row keeps the columns the
// shared state machine reads; delivery is the only mode-dependent part.
type request struct {
	row      db.DeviceAuthorization
	delivery delivery
}

// parseRequest is the only way a row becomes a request. client_id decides the
// mode. A row whose redirect columns disagree with client_id is corrupt and
// fails here rather than deeper in the state machine.
func parseRequest(row db.DeviceAuthorization) (request, error) {
	if !row.ClientID.Valid {
		if row.RedirectUri.Valid || row.RedirectUriPresented.Valid || row.State.Valid || row.CodeChallenge.Valid || row.CodeHash.Valid {
			return request{}, corruptRequest(row.ID)
		}
		return request{row: row, delivery: deviceDelivery{}}, nil
	}
	if !row.RedirectUri.Valid || !row.RedirectUriPresented.Valid || !row.CodeChallenge.Valid {
		return request{}, corruptRequest(row.ID)
	}
	return request{row: row, delivery: redirectDelivery{
		clientID: ClientID(row.ClientID.String),
		redirectURI: RedirectURI{
			raw:       row.RedirectUri.String,
			presented: row.RedirectUriPresented.Bool,
		},
		state:     row.State.String,
		challenge: Challenge{value: row.CodeChallenge.String},
	}}, nil
}

func corruptRequest(id string) error {
	return fault.New("grant request columns disagree with its delivery mode",
		fault.Internal("device_authorizations row "+id+" has inconsistent oauth columns"),
	)
}

func hashCode(code string) string {
	return hash.Sha256(code)
}
