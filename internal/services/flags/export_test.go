package flags

import "context"

const HelloWorldSlug = "hello_world"

func (s *service) IsHelloWorldEnabled(ctx context.Context, workspaceID string) bool {
	return s.enabled(ctx, workspaceID, HelloWorldSlug, false)
}
