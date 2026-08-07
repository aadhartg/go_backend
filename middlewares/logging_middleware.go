// middlewares/logging_middleware.go
package middlewares

import (
	"log"
	"net/http"
	"strings"
	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// LoggingMiddleware logs each incoming HTTP request with details such as method, URI, status code, and duration.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Capture the start time
		start := time.Now()

		// Process the request
		c.Next()

		// Calculate the duration
		duration := time.Since(start)

		// Log the details
		log.Printf(
			"%s %s %d %s",
			c.Request.Method,
			c.Request.RequestURI,
			c.Writer.Status(),
			duration,
		)

		// Log errors if the status code indicates an error
		if c.Writer.Status() >= 400 {
			logError(config.DB, c, c.Writer.Status(), duration)
		}
	}
}

// logError logs and saves the error to the database.
func logError(db *gorm.DB, c *gin.Context, statusCode int, duration time.Duration) {
	errorType := "general"

	// Determine specific error types for known routes
	if strings.Contains(c.Request.URL.Path, utils.RouteProductPaymentDetails) || strings.Contains(c.Request.URL.Path, utils.RoutePaymentSuccessPaypal) {
		errorType = "payment"
	}

	// Get current time for logging
	now := time.Now()

	// Attempt to find an existing log entry
	var logEntry models.ErrorLog
	result := db.Where("api_path = ?", c.Request.URL.Path).First(&logEntry)

	if result.Error == nil {
		// Update occurrence count and last occurred time
		db.Model(&logEntry).Updates(models.ErrorLog{
			OccurrenceCount: logEntry.OccurrenceCount + 1,
			LastOccurredAt:  now,
		})
	} else if result.Error == gorm.ErrRecordNotFound {
		// Insert a new error log entry
		newLog := models.ErrorLog{
			APIPath:         c.Request.URL.Path,
			ErrorMessage:    http.StatusText(statusCode),
			ErrorType:       errorType,
			OccurrenceCount: 1,
			LastOccurredAt:  now,
		}
		db.Create(&newLog)
	} else {
		// Handle other potential errors from the database query
		log.Printf("Error querying error logs: %v", result.Error)
	}
}
