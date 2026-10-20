package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/response"
)

type bucket struct {
	tokens int
	reset  time.Time
}

// RateLimit 简单的按 IP 限流（固定窗口），用于登录防爆破。
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := make(map[string]*bucket)

	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()

		mu.Lock()
		b, ok := buckets[key]
		if !ok || now.After(b.reset) {
			b = &bucket{tokens: limit, reset: now.Add(window)}
			buckets[key] = b
		}
		b.tokens--
		remain := b.tokens
		if len(buckets) > 10000 {
			for k, v := range buckets {
				if now.After(v.reset) {
					delete(buckets, k)
				}
			}
		}
		mu.Unlock()

		if remain < 0 {
			c.Header("Retry-After", window.String())
			response.Abort(c, 429, errcode.TooManyReqs, "")
			return
		}
		c.Next()
	}
}
