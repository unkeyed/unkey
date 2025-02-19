package entities

import "time"

type RatelimitNamespace struct {
	ID          string
	WorkspaceID string
	Name        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   time.Time
}
