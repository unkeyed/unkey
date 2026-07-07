package stripe

import (
	"fmt"
	"strings"

	stripesdk "github.com/stripe/stripe-go/v86"
)

// NewClient builds a Stripe client for dev tools, refusing live keys.
func NewClient(key string) (*stripesdk.Client, error) {
	if key == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY is not set")
	}
	if !strings.HasPrefix(key, "sk_test_") && !strings.HasPrefix(key, "rk_test_") {
		return nil, fmt.Errorf("refusing to run: the Stripe key is not a test-mode key")
	}

	return stripesdk.NewClient(key, stripesdk.WithBackends(stripesdk.NewBackendsWithConfig(&stripesdk.BackendConfig{
		//nolint:exhaustruct // defaults are fine except the logger
		LeveledLogger: &stripesdk.LeveledLogger{Level: stripesdk.LevelNull},
	}))), nil
}
