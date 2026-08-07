// seeds/roles_seeder.go
package seeds

import (
	"log"

	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"gorm.io/gorm"
)

// SeedRoles inserts predefined roles into the roles table.
// It ensures that roles are not duplicated by checking for existing entries.
func SeedRoles() {
	// Define the roles to be seeded
	roles := []models.Role{
		{Name: "super-admin"},
		{Name: "admin"},
		{Name: "user"},
		{Name: "patient"},
		{Name: "coordinator"},
	}

	for _, role := range roles {
		// Check if the role already exists
		var existingRole models.Role
		result := config.DB.Where("name = ?", role.Name).First(&existingRole)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// Role does not exist, create it
				if err := config.DB.Create(&role).Error; err != nil {
					log.Printf(utils.MsgFailedToCreateRole, role.Name, err)
				} else {
					log.Printf(utils.MsgRoleCreated, role.Name)
				}
			} else {
				// An unexpected error occurred
				log.Printf(utils.MsgFailedToCheckRole, role.Name, result.Error)
			}
		} else {
			// Role already exists
			log.Printf(utils.MsgRoleAlreadyExists, role.Name)
		}
	}

	log.Println(utils.MsgRolesSeededSuccessfully)
}
