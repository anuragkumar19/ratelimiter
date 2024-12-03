package ratelimiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

type BucketCtx struct {
	ID                 string    `json:"id"`
	Revision           int       `json:"revision"`
	LastResetAt        time.Time `json:"last_reset_at"`
	ConsumedTokenCount int       `json:"consumed_token_count"`
	LastConsumedAt     time.Time `json:"last_consumed_at"`
}

type Bucket[T any] struct {
	mu  sync.Mutex
	l   *Limiter[T]
	ctx *BucketCtx
}

func (b *Bucket[T]) Ctx() BucketCtx {
	b.mu.Lock()
	defer b.mu.Unlock()
	return *b.ctx
}

func (b *Bucket[T]) Consume(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()

	if b.ctx.LastResetAt.Add(b.l.resetAfter).Before(now) {
		b.ctx.ConsumedTokenCount = 0
		b.ctx.LastResetAt = now
		b.ctx.LastConsumedAt = time.Time{}
	}

	if b.ctx.ConsumedTokenCount >= b.l.limit {
		tryAfter := b.ctx.LastResetAt.Add(b.l.resetAfter)
		return &RateLimitError{
			Remaining: 0,
			ResetAt:   tryAfter,
			TryAfter:  tryAfter,
		}
	}

	backoffLen := len(b.l.backOffs)
	if backoffLen > 0 && b.ctx.ConsumedTokenCount > 0 {
		i := b.ctx.ConsumedTokenCount - 1
		if b.ctx.ConsumedTokenCount > backoffLen {
			i = backoffLen - 1
		}
		d := b.l.backOffs[i]

		tryAfter := b.ctx.LastConsumedAt.Add(d)
		resetAt := b.ctx.LastResetAt.Add(b.l.resetAfter)
		if resetAt.Before(tryAfter) {
			tryAfter = resetAt
		}
		if tryAfter.After(now) {
			return &RateLimitError{
				Remaining: b.l.limit - b.ctx.ConsumedTokenCount,
				ResetAt:   resetAt,
				TryAfter:  tryAfter,
			}
		}
	}

	b.ctx.ConsumedTokenCount++
	b.ctx.LastConsumedAt = now

	if err := b.l.store.Update(ctx, *b.ctx); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrRevisionMismatch
		}
		return err
	}
	return nil
}
