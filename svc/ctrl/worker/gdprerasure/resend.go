package gdprerasure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// resendBaseURL is the Resend REST API origin.
const resendBaseURL = "https://api.resend.com"

// ResendContacts deletes the Resend contacts the dashboard adds for its users.
type ResendContacts interface {
	// DeleteContact deletes the contact with this email address and reports
	// whether it existed. A contact that is already gone counts as deleted.
	DeleteContact(ctx context.Context, email string) (bool, error)
}

type resendContacts struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

var _ ResendContacts = (*resendContacts)(nil)

// NewResend builds a Resend-backed ResendContacts from an API key. The key
// needs full access; a sending-only key cannot delete contacts.
func NewResend(apiKey string) ResendContacts {
	return &resendContacts{
		apiKey:  apiKey,
		baseURL: resendBaseURL,
		client:  &http.Client{Timeout: 10 * time.Second}, //nolint:exhaustruct // default transport, only the timeout matters
	}
}

// DeleteContact uses the team-wide contacts endpoint. Resend keeps one contact
// per email across all audiences and segments, so this removes the address
// everywhere, not only from the audience the dashboard adds it to.
func (r *resendContacts) DeleteContact(ctx context.Context, email string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.baseURL+"/contacts/"+url.PathEscape(email), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	res, err := r.client.Do(req)
	if uErr, ok := errors.AsType[*url.Error](err); ok {
		// The URL holds the address being erased; keep it out of error logs.
		return false, fmt.Errorf("resend DELETE /contacts: %w", uErr.Err)
	}
	if err != nil {
		return false, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return false, fmt.Errorf("resend DELETE /contacts: status %d: %s", res.StatusCode, body)
	}
	return true, nil
}
