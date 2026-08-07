// services/log_cleanup_service.go
package services

import (
	"theransticslabs/m/models"
	"time"

	"gorm.io/gorm"
)

func LogCleanupJob(db *gorm.DB) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		cleanupOldLogs(db)
	}
}

func cleanupOldLogs(db *gorm.DB) {
	threshold := time.Now().AddDate(0, -1, 0) // One month ago
	db.Where("created_at < ?", threshold).Delete(&models.ErrorLog{})
}
