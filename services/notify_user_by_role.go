// services/notify_user_by_role.go
package services

import (
	"fmt"
	"theransticslabs/m/models"

	"gorm.io/gorm"
)

// NotifyUsersByRoles notifies all users with the specified roles.
func NotifyUsersByRoles(db *gorm.DB, tx *gorm.DB, roleNames []string, title string, notificationType string, messageTemplate string, notificationUsername string, entityType string, entityID uint, metadata interface{}) error {
	// Use transaction if provided, otherwise fallback to `db`
	dbInstance := db
	if tx != nil {
		dbInstance = tx
	}

	// Fetch all roles by names
	var roles []models.Role
	if err := dbInstance.Where("name IN (?) AND is_deleted = ?", roleNames, false).Find(&roles).Error; err != nil {
		return fmt.Errorf("failed to fetch roles: %v", err)
	}

	if len(roles) == 0 {
		return fmt.Errorf("no matching roles found for: %v", roleNames)
	}

	// Map role IDs
	roleIDs := make([]uint, 0, len(roles))
	for _, role := range roles {
		roleIDs = append(roleIDs, role.ID)
	}

	// Fetch all users with the specified roles
	var users []models.User
	if err := dbInstance.Where("role_id IN (?)", roleIDs).Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users for roles: %v", err)
	}

	// Create notifications for each user
	for _, user := range users {

		err := CreateNotification(
			dbInstance,
			user.ID,          // Notify the user
			title,            // Title
			messageTemplate,  // Formatted Message
			notificationType, // notification type
			notificationUsername,
			entityType, // Entity Type
			entityID,   // Entity ID
			metadata,
		)
		if err != nil {
			return fmt.Errorf("failed to send notification to user '%s %s': %v", user.FirstName, user.LastName, err)
		}
	}

	return nil
}
