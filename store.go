package ratelimiter

import (
	"context"
	"errors"
)

var ErrRevisionMismatch = errors.New("bucket revision mismatch")
var ErrNotFound = errors.New("bucket not found")
var ErrAlreadyExist = errors.New("bucket already exist")

type Store interface {
	Get(ctx context.Context, id string) (BucketCtx, error)
	Create(context.Context, BucketCtx) error
	Update(context.Context, BucketCtx) error
}
