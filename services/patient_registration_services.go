// services/patient_registration_services.go

package services

import (
	"errors"
	"fmt"
	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"gorm.io/gorm"
)

func ValidatePatientRequestData(firstName string, lastName string, email string) error {
	// First name validation
	if !utils.IsValidFirstName(firstName) {
		return fmt.Errorf(utils.MsgInvalidFirstName)
	}

	// Last name validation (optional)
	if lastName != "" && !utils.IsValidLastName(lastName) {
		return fmt.Errorf(utils.MsgInvalidLastName)
	}

	if !utils.IsValidEmail(email) {
		return fmt.Errorf(utils.MsgInvalidEmail)
	}

	// Email uniqueness check
	var existingUser models.User
	if err := config.DB.Where("email = ?", email).First(&existingUser).Error; err != gorm.ErrRecordNotFound {
		return errors.New("Email is already registered")
	}

	// NHI validation

	// if role == "patient" {
	// 	if strings.TrimSpace(nhi) == "" {
	// 		return errors.New("NHI number is required")
	// 	}

	// 	if !utils.IsValidNHI(nhi) {
	// 		return errors.New("NHI number must contain 3 letters followed by 4 digits (e.g. ABC1234)")
	// 	}

	// 	var existingNHI models.User
	// 	if err := config.DB.Where("nhi = ?", nhi).First(&existingNHI).Error; err == nil {
	// 		return errors.New("NHI number is already registered")
	// 	}
	// }
	
	return nil
}

// InsertPatientRegistration creates a new user with patient role
func InsertPatientRegistration(tx *gorm.DB, firstName, lastName, email string,  roleName string) (*models.User, error) {
	var role models.Role
	if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
		//	return nil, errors.New("Role 'patient' not found")
		return nil, fmt.Errorf("role '%s' not found", roleName)
	}

	user := &models.User{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		RoleID:    role.ID,
	}

	if err := tx.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}
