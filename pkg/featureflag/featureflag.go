package featureflag

import (
	"context"
	"errors"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/isolated"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// PrivateNetworking gates private networking activation. The dashboard
// declares the same key in web/apps/dashboard/lib/flags/index.ts.
const PrivateNetworking = "private-networking"

// New creates an isolated OpenFeature API and installs provider. A nil provider
// leaves the API's default-only no-op provider installed.
func New(ctx context.Context, provider openfeature.FeatureProvider) (*openfeature.EvaluationAPI, error) {
	api := isolated.NewAPI()
	if provider == nil {
		return api, nil
	}
	if err := api.SetProviderAndWait(ctx, provider); err != nil {
		return nil, errors.Join(err, api.Shutdown(context.WithoutCancel(ctx)))
	}
	return api, nil
}

// TeamContext returns the evaluation context for a team. orgID is the WorkOS
// organization ID, stored as workspaces.org_id, which the dashboard also sends
// as team.id.
func TeamContext(orgID string) openfeature.EvaluationContext {
	return openfeature.NewTargetlessEvaluationContext(map[string]any{
		"team": map[string]any{"id": orgID},
	})
}

// Static returns a provider that resolves each listed boolean flag to its
// value for every context. Unlisted flags resolve with FLAG_NOT_FOUND.
func Static(values map[string]bool) openfeature.FeatureProvider {
	flags := make(map[string]memprovider.InMemoryFlag, len(values))
	for key, value := range values {
		variant := "off"
		if value {
			variant = "on"
		}
		flags[key] = memprovider.InMemoryFlag{
			Key:              key,
			State:            memprovider.Enabled,
			DefaultVariant:   variant,
			Variants:         map[string]any{"off": false, "on": true},
			ContextEvaluator: nil,
		}
	}
	return memprovider.NewInMemoryProvider(flags)
}
