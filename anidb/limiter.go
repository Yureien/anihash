package anidb

import (
	"context"

	"golang.org/x/time/rate"
)

// A Limiter is a rate limiter that complies with AniDB UDP API flood
// prevention recommendations.
//
// It functions similarly to [golang.org/x/time/rate.Limiter], except
// with both short and long term limits.
type limiter struct {
	short *rate.Limiter
	long  *rate.Limiter
}

func newLimiter() *limiter {
	return &limiter{
		// Every 2 sec short term
		short: rate.NewLimiter(0.5, 1),
		// Every 4 sec long term after 60 seconds
		long: rate.NewLimiter(0.25, 60/2),
	}
}

func (l limiter) Wait(ctx context.Context) error {
	if err := l.long.Wait(ctx); err != nil {
		return err
	}
	if err := l.short.Wait(ctx); err != nil {
		return err
	}
	return nil
}
