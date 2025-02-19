package transform

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
)

func WorkspaceModelToEntity(m gen.Workspace) (entities.Workspace, error) {
	workspace := entities.Workspace{
		ID:                   m.ID,
		TenantID:             m.TenantID,
		Name:                 m.Name,
		Enabled:              m.Enabled,
		DeleteProtection:     true,
		CreatedAt:            time.Time{},
		DeletedAt:            time.Time{},
		Plan:                 entities.WorkspacePlan(m.Plan.WorkspacesPlan),
		StripeCustomerID:     "",
		StripeSubscriptionID: "",
		TrialEnds:            time.Time{},
		PlanLockedUntil:      time.Time{},
		BetaFeatures:         map[string]any{},
		Features:             map[string]any{},
	}

	if m.CreatedAt.Valid {
		workspace.CreatedAt = m.CreatedAt.Time
	}

	if m.DeletedAt.Valid {
		workspace.DeletedAt = m.DeletedAt.Time
	}

	if m.Plan.Valid {
		workspace.Plan = entities.WorkspacePlan(m.Plan.WorkspacesPlan)
	} else {
		workspace.Plan = entities.WorkspacePlanFree
	}

	if m.DeleteProtection.Valid {
		workspace.DeleteProtection = m.DeleteProtection.Bool
	}

	if m.StripeCustomerID.Valid {
		workspace.StripeCustomerID = m.StripeCustomerID.String
	}

	if m.StripeSubscriptionID.Valid {
		workspace.StripeSubscriptionID = m.StripeSubscriptionID.String
	}

	if m.TrialEnds.Valid {
		workspace.TrialEnds = m.TrialEnds.Time
	}

	if m.PlanLockedUntil.Valid {
		workspace.PlanLockedUntil = m.PlanLockedUntil.Time
	}

	if err := json.Unmarshal(m.BetaFeatures, &workspace.BetaFeatures); err != nil {
		return entities.Workspace{}, fmt.Errorf("unable to unmarshal beta features: %w", err)
	}

	if err := json.Unmarshal(m.Features, &workspace.Features); err != nil {
		return entities.Workspace{}, fmt.Errorf("unable to unmarshal features: %w", err)
	}

	return workspace, nil
}
