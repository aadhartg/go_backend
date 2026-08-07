package middlewares

import (
	"net/http"
	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"time"

	"github.com/gin-gonic/gin"
)

func ConsentSessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		token := c.GetHeader("X-Consent-Token")

		if token == "" {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"message": "Consent session token required"},
			)
			return
		}

		var session models.ConsentSession

		err := config.DB.
			Where("token = ? AND is_active = ?", token, true).
			First(&session).Error

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"message": "Invalid session"},
			)
			return
		}

		if session.ExpiresAt.Before(time.Now()) {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"message": "Session expired"},
			)
			return
		}

		c.Set("consentSession", session)

		c.Next()
	}
}

func GetConsentSession(
	c *gin.Context,
) (*models.ConsentSession, bool) {

	session, exists := c.Get("consentSession")

	if !exists {
		return nil, false
	}

	consentSession, ok :=
		session.(models.ConsentSession)

	if !ok {
		return nil, false
	}

	return &consentSession, true
}
