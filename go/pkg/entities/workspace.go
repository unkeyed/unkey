package entities

import "time"

type WorkspacePlan string

const (
	WorkspacePlanFree       WorkspacePlan = "free"
	WorkspacePlanPro        WorkspacePlan = "pro"
	WorkspacePlanEnterprise WorkspacePlan = "enterprise"
)

type Workspace struct {
	ID                   string
	TenantID             string
	Name                 string
	CreatedAt            time.Time
	DeletedAt            time.Time
	Plan                 WorkspacePlan
	Enabled              bool
	DeleteProtection     bool
	BetaFeatures         map[string]interface{}
	Features             map[string]interface{}
	StripeCustomerID     string
	StripeSubscriptionID string
	TrialEnds            time.Time
	PlanLockedUntil      time.Time
}
