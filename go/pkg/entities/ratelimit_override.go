package entities

import (
	"time"
)

type RatelimitOverride struct {
	ID          string
	WorkspaceID string
	NamespaceID string
	Identifier  string
	Limit       int32
	Duration    time.Duration
	Async       bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   time.Time
}
