package ratelimit

import (
	"errors"
	"fmt"
	"time"

	"nuistagram/internal/cache"
)

var ErrLimitExceeded = errors.New("rate limit exceeded")

// dailyActions are those tracked per calendar day.
var dailyActions = map[string]bool{
	"upload": true,
	"avatar": true,
	"export": true,
}

type Limiter struct {
	daily  *cache.Cache
	hourly *cache.Cache
}

func New() *Limiter {
	return &Limiter{
		daily:  cache.New(25 * time.Hour),
		hourly: cache.New(2 * time.Hour),
	}
}

// CheckAndIncrement checks whether userID has exceeded limit for action,
// and increments the counter if not. A limit of 0 means unlimited.
// action must be one of: "upload", "avatar", "export", "comment", "like".
func (l *Limiter) CheckAndIncrement(userID int64, action string, limit int) error {
	if limit == 0 {
		return nil
	}

	var key string
	var c *cache.Cache

	if dailyActions[action] {
		key = fmt.Sprintf("rl:%s:%d:%s", action, userID, time.Now().UTC().Format("2006-01-02"))
		c = l.daily
	} else {
		key = fmt.Sprintf("rl:%s:%d:%s", action, userID, time.Now().UTC().Format("2006-01-02-15"))
		c = l.hourly
	}

	count := 0
	if val, ok := c.Get(key); ok {
		count = val.(int)
	}

	if count >= limit {
		return ErrLimitExceeded
	}

	c.Set(key, count+1)
	return nil
}
