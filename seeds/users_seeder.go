// seeds/users_seeder.go
package seeds

import (
	"log"

	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"gorm.io/gorm"
)

// SeedUsers inserts predefined users into the users table.
// It ensures that users are not duplicated by checking for existing entries.
func SeedUsers() {
	// Define the users to be seeded
	users := []struct {
		FirstName string
		LastName  string
		Email     string
		RoleName  string
	}{
		{
			FirstName: "Client",
			LastName:  "SuperAdmin",
			Email:     "dna.client.superadmin@yopmail.com",
			RoleName:  "super-admin",
		},
		{
			FirstName: "Dev",
			LastName:  "SuperAdmin",
			Email:     "dna.dev.superadmin@yopmail.com",
			RoleName:  "super-admin",
		},
		{
			FirstName: "Test",
			LastName:  "SuperAdmin",
			Email:     "dna.test.superadmin@yopmail.com",
			RoleName:  "super-admin",
		},
		{
			FirstName: "Client",
			LastName:  "Admin",
			Email:     "dna.client.admin@yopmail.com",
			RoleName:  "admin",
		},
		{
			FirstName: "Dev",
			LastName:  "Admin",
			Email:     "dna.dev.admin@yopmail.com",
			RoleName:  "admin",
		},
		{
			FirstName: "Test",
			LastName:  "Admin",
			Email:     "dna.test.admin@yopmail.com",
			RoleName:  "admin",
		},
		{
			FirstName: "Client",
			LastName:  "User",
			Email:     "dna.client.user@yopmail.com",
			RoleName:  "user",
		},
		{
			FirstName: "Dev",
			LastName:  "User",
			Email:     "dna.dev.user@yopmail.com",
			RoleName:  "user",
		},
		{
			FirstName: "Test",
			LastName:  "User",
			Email:     "dna.test.user@yopmail.com",
			RoleName:  "user",
		},
	}

	for _, u := range users {
		// Check if the user already exists by  email
		var existingUser models.User
		result := config.DB.Where(" email = ?", u.Email).First(&existingUser)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// User does not exist, proceed to create
				// Fetch the role ID based on RoleName
				var role models.Role
				roleResult := config.DB.Where("name = ?", u.RoleName).First(&role)
				if roleResult.Error != nil {
					log.Printf("Error fetching role '%s' for user '%s': %v", u.RoleName, u.Email, roleResult.Error)
					continue
				}

				// Create the user
				user := models.User{
					FirstName: u.FirstName,
					LastName:  u.LastName,
					Email:     u.Email,
					RoleID:    role.ID,
				}

				if err := config.DB.Create(&user).Error; err != nil {
					log.Printf(utils.MsgFailedToCreateUser, u.Email, err)
				} else {
					log.Printf(utils.MsgUserCreated, u.Email)
				}
			} else {
				// An unexpected error occurred
				log.Printf(utils.MsgFailedToCheckUser, u.Email, result.Error)
			}
		} else {
			// User already exists
			log.Printf(utils.MsgUserAlreadyExists, u.Email)
		}
	}

	log.Println(utils.MsgUsersSeededSuccessfully)
}
