package ratelimiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

type HashFunc[T any] func(T) string

type Limiter[T any] struct {
	label      string
	hashFunc   HashFunc[T]
	resetAfter time.Duration
	backOffs   []time.Duration
	store      Store
	limit      int
}

type LimiterOption[T any] struct {
	Label      string
	Limit      int
	ResetAfter time.Duration
	HashFunc   HashFunc[T]
	Store      Store
	BackOffs   []time.Duration
}

func New[T any](option *LimiterOption[T]) (*Limiter[T], error) {
	if option.Label == "" {
		return nil, errors.New("label cannot be empty")
	}
	if option.Limit < 0 {
		return nil, errors.New("limit cannot be negative")
	}
	if option.ResetAfter <= 0 {
		return nil, errors.New("resetAfter cannot be zero or negative")
	}
	if option.HashFunc == nil {
		return nil, errors.New("hashFunc cannot be nil")
	}
	if len(option.BackOffs) >= option.Limit {
		return nil, errors.New("backOffs length must be less than limit")
	}
	if option.Store == nil {
		return nil, errors.New("store cannot be nil")
	}

	return &Limiter[T]{
		label:      option.Label,
		hashFunc:   option.HashFunc,
		resetAfter: option.ResetAfter,
		backOffs:   option.BackOffs,
		store:      option.Store,
		limit:      option.Limit,
	}, nil
}

func (l *Limiter[T]) Bucket(ctx context.Context, v T) (*Bucket[T], error) {
	id := l.label + ":" + l.hashFunc(v)
	b, err := l.store.Get(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}

		newB := BucketCtx{
			ID:                 id,
			Revision:           1,
			LastResetAt:        time.Now(),
			ConsumedTokenCount: 0,
			LastConsumedAt:     time.Time{},
		}

		if err := l.store.Create(ctx, newB); err != nil {
			if errors.Is(err, ErrAlreadyExist) {
				return nil, ErrRevisionMismatch
			}
		}
		b = newB
	}

	return &Bucket[T]{
		mu:  sync.Mutex{},
		l:   l,
		ctx: &b,
	}, nil
}

func String(s string) string {
	return s
}
