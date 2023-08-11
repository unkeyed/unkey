package events

import (
	"context"
)

type noop struct{}

func (n *noop) EmitKeyEvent(ctx context.Context, event KeyEvent)           {}
func (n *noop) OnKeyEvent(func(ctx context.Context, event KeyEvent) error) {}

func NewNoop() EventBus {
	return &noop{}
}
