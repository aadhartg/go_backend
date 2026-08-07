// services/notification_service.go
package services

import (
	"encoding/json"
	"log"
	"theransticslabs/m/models"

	"gorm.io/gorm"
)

// CreateNotification stores a notification in the database
func CreateNotification(db *gorm.DB, userID uint, title, message, notificationType, notificationUsername, entityType string, entityID uint, metadata interface{}) error {

	// Convert metadata to json.RawMessage
	var metadataJSON json.RawMessage
	if metadata != nil {
		jsonBytes, err := json.Marshal(metadata)
		if err != nil {
			log.Printf("Failed to marshal metadata: %v", err)
			return err
		}
		metadataJSON = jsonBytes
	}

	notification := models.Notification{
		UserID:     userID,
		Title:      title,
		Message:    message,
		Type:       notificationType,
		EntityType: entityType,
		EntityID:   entityID,
		Username:   notificationUsername,
		Metadata:   metadataJSON,
	}

	// Insert notification into the database
	if err := db.Create(&notification).Error; err != nil {
		log.Printf("Failed to create notification: %v", err)
		return err
	}

	log.Printf("Notification created: %v", notification)
	return nil
}
