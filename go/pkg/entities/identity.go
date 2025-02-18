package entities

import "time"

type Identity struct {
	ID          string
	ExternalID  string
	WorkspaceID string
	Meta        map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   time.Time
	Environment string
	// Ratelimits
}
