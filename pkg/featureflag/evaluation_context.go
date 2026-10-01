package featureflag

import "github.com/open-feature/go-sdk/openfeature"

// TeamContext returns the evaluation context for a team. orgID is the WorkOS
// organization ID, stored as workspaces.org_id, which the dashboard also sends
// as team.id.
func TeamContext(orgID string) openfeature.EvaluationContext {
	return openfeature.NewTargetlessEvaluationContext(map[string]any{
		"team": map[string]any{"id": orgID},
	})
}
