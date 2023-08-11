package events

import (
	"context"
)

type EventBus interface {
	EmitKeyEvent(ctx context.Context, event KeyEvent)
	OnKeyEvent(func(ctx context.Context, event KeyEvent) error)
}

type KeyEventType string

var (
	KeyCreated KeyEventType = "created"
	KeyUpdated KeyEventType = "updated"
	KeyDeleted KeyEventType = "deleted"
)

type Key struct {
	Id   string `json:"id"`
	Hash string `hson:"hash"`
}

type KeyEvent struct {
	Type KeyEventType `json:"type"`
	Key  Key          `json:"key"`
}
