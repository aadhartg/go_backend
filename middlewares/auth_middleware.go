// middlewares/auth_middleware.go
package middlewares

import (
	"net/http"
	"strings"

	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Define a type for context keys to avoid collisions
type contextKey string

const userContextKey = contextKey("user")

// AuthMiddleware verifies the JWT token, ensures the user exists and is active,
// and sets the user information in the request context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgAuthHeaderMissing, nil)
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgInvalidAuthHeaderFormat, nil)
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate JWT token
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgInvalidOrExpiredToken, nil)
			c.Abort()
			return
		}

		// Extract user information from claims
		userID, ok := claims["id"]
		if !ok {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgInvalidTokenClaims, nil)
			c.Abort()
			return
		}

		// Fetch the user from the database
		var user models.User
		result := config.DB.Preload("Role").First(&user, userID)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserDoesNotExist, nil)
				c.Abort()
				return
			}
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
			c.Abort()
			return
		}

		// Check if user is active and not deleted
		if !user.ActiveStatus || user.IsDeleted {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserAccountInactiveOrDeleted, nil)
			c.Abort()
			return
		}

		// Compare the token in the request with the one stored in the database
		if user.Token != tokenString {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUnauthorizedUser, nil)
			c.Abort()
			return
		}

		// Set user information in the context
		c.Set(string(userContextKey), user)
		c.Next()
	}
}

// GetUserFromContext retrieves the user information from the context.
// It returns the user and a boolean indicating whether the user was found.
func GetUserFromContext(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get(string(userContextKey))
	if !exists {
		return nil, false
	}
	userModel, ok := user.(models.User)
	if !ok {
		return nil, false
	}
	return &userModel, true
}
