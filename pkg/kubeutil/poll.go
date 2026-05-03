package kubeutil

import (
	"context"
	"time"
)

// pollUntil calls fn immediately, then every interval, until fn returns (true, nil),
// ctx is cancelled, or a tick after ctx cancels.
//
// fn returning (false, nil) means "not ready yet — keep retrying".
// fn returning (false, non-nil) is treated the same as (false, nil): log and retry.
// fn returning (true, nil) terminates with success.
// ctx cancellation terminates with ctx.Err().
func pollUntil(ctx context.Context, interval time.Duration, fn func() (bool, error)) error {
	if ok, _ := fn(); ok {
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if ok, _ := fn(); ok {
				return nil
			}
		}
	}
}
