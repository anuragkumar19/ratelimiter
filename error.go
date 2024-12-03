package ratelimiter

import (
	"fmt"
	"time"
)

type RateLimitError struct {
	Remaining int
	ResetAt   time.Time
	TryAfter  time.Time
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("too many requests, try after %s", time.Until(e.TryAfter).String())
}
