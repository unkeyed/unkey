package notification

import restate "github.com/restatedev/sdk-go"

type deliveryRef struct {
	WorkspaceID string `json:"workspaceId"`
	ID          string `json:"id"`
}

// Deliver retries one frozen target and returns its final status.
// Ambiguous sends stop unless the provider's idempotency window permits a retry.
func (s *Service) Deliver(ctx restate.ObjectContext, ref deliveryRef) (string, error) {
	if restate.Key(ctx) != ref.ID || ref.WorkspaceID == "" {
		return "", invalid("invalid delivery key")
	}
	for {
		result, err := restate.Run(ctx, func(runCtx restate.RunContext) (attemptResult, error) {
			return s.attempt(runCtx, ref)
		}, restate.WithName("attempt notification delivery"))
		if err != nil {
			return "", err
		}
		if result.RetryAfter <= 0 {
			return string(result.Status), nil
		}
		if err := restate.Sleep(ctx, result.RetryAfter); err != nil {
			return "", err
		}
	}
}
