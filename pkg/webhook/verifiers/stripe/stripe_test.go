package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// sign produces a Stripe-Signature header for body: t=<ts>,v1=<hmac-sha256
// of "<ts>.<body>">, the scheme ConstructEvent validates.
func sign(secret, body string, ts time.Time) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(fmt.Appendf(nil, "%d.%s", ts.Unix(), body))
	return fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifier(t *testing.T) {
	const secret = "whsec_test"
	// "object":"event" matters: stripe-go v21+ rejects payloads without it as
	// thin event notifications, which need a different parser.
	body := `{"id":"evt_123","object":"event","type":"invoice.created","data":{"object":{"id":"in_123"}}}`

	request := func(signature string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(body))
		r.Header.Set("Stripe-Signature", signature)
		return r
	}

	t.Run("valid signature yields the unwrapped event", func(t *testing.T) {
		event, err := New(secret).Verify(request(sign(secret, body, time.Now())))
		require.NoError(t, err)
		require.Equal(t, "evt_123", event.ID)
		require.Equal(t, "invoice.created", event.Type)
		require.JSONEq(t, `{"id":"in_123"}`, string(event.Payload))
	})

	t.Run("wrong secret is rejected", func(t *testing.T) {
		_, err := New(secret).Verify(request(sign("whsec_other", body, time.Now())))
		require.Error(t, err)
	})

	t.Run("stale timestamp is rejected", func(t *testing.T) {
		_, err := New(secret).Verify(
			request(sign(secret, body, time.Now().Add(-time.Hour))))
		require.Error(t, err)
	})
}
