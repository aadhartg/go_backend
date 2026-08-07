package middlewares

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"theransticslabs/m/config"
)

func RateLimit(limit int, duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {

		ip := c.ClientIP()

		key := fmt.Sprintf("rate_limit:%s:%s", c.FullPath(), ip)

		count, err := config.RedisClient.Incr(config.Ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			config.RedisClient.Expire(config.Ctx, key, duration)
		}

		if count > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "Too many requests. Please try again after some time.",
			})
			return
		}

		c.Next()
	}
}
