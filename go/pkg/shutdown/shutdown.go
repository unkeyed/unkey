package shutdown

import "context"

type ShutdownFn func(ctx context.Context) error
