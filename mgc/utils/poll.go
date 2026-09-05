package utils

import (
	"context"
	"time"
)

// Poll checks immediately, then waits between attempts. A deadline or
// cancellation is always an error, never a successful resource transition.
func Poll(ctx context.Context, timeout, interval time.Duration, check func(context.Context) (bool, error)) error {
	return poll(ctx, timeout, interval, false, check)
}

// PollAfter preserves an initial settling interval for operations whose API
// can briefly report the pre-operation status after accepting a request.
func PollAfter(ctx context.Context, timeout, interval time.Duration, check func(context.Context) (bool, error)) error {
	return poll(ctx, timeout, interval, true, check)
}

func poll(ctx context.Context, timeout, interval time.Duration, delayFirst bool, check func(context.Context) (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !delayFirst {
			done, err := check(ctx)
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if done {
				return nil
			}
		}
		delayFirst = false
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
