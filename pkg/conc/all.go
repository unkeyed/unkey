package conc

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"golang.org/x/sync/errgroup"
)

// All runs fns at the same time and waits for them, like Promise.all. It
// returns the first error and cancels the ctx of the other fns. A panic in a fn
// becomes an error, because an unrecovered panic in a goroutine stops the process
//
//	var user User
//	var orders []Order
//	err := conc.All(ctx,
//	    func(ctx context.Context) (err error) {
//	        user, err = findUser(ctx, id)
//	        return err
//	    },
//	    func(ctx context.Context) (err error) {
//	        orders, err = listOrders(ctx, id)
//	        return err
//	    },
//	)
func All(ctx context.Context, fns ...func(ctx context.Context) error) error {
	g, gctx := errgroup.WithContext(ctx)
	for _, fn := range fns {
		g.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fault.New("panic in concurrent call",
						fault.Code(codes.App.Internal.UnexpectedError.URN()),
						fault.Internal(fmt.Sprintf("panic: %v\n%s", r, debug.Stack())),
						fault.Public("An unexpected error occurred while processing your request."),
					)
				}
			}()

			return fn(gctx)
		})
	}
	return g.Wait()
}
