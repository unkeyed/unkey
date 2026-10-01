package featureflag

import (
	"context"
	"errors"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/isolated"
)

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
