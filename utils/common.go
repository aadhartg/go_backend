// utils/common.go
package utils

import (
	"fmt"
	"strings"
	"theransticslabs/m/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindUserByEmail searches for a user by email.
// Returns the user and any error encountered.
func FindUserByEmail(db *gorm.DB, email string) (*models.User, error) {
	var user models.User
	// Corrected query with two conditions
	result := db.Preload("Role").Where("email = ? AND is_deleted = ?", email, false).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// StringInSlice checks if a string exists in a slice of strings
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// GenerateUniqueOrderID generates a unique order ID in the format SL-YYYYMMDD-XXXX
func GenerateUniqueOrderID(db *gorm.DB) (string, error) {
	const prefix = "SL"
	const maxAttempts = 100 // Maximum attempts to generate a unique ID

	// Get the current date in YYYYMMDD format
	currentDate := time.Now().Format("20060102")

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Generate a random 4-digit number from UUID
		randomSuffix := uuid.New().ID() % 10000

		// Format the order ID
		orderID := fmt.Sprintf("%s-%s-%04d", prefix, currentDate, randomSuffix)

		// Check if the order ID exists in the database
		var count int64
		if err := db.Model(&models.Order{}).Where("order_number = ?", orderID).Count(&count).Error; err != nil {
			return "", fmt.Errorf("failed to check order ID uniqueness: %v", err)
		}

		// If the order ID is unique, return it
		if count == 0 {
			return orderID, nil
		}
	}

	return "", fmt.Errorf("failed to generate a unique order ID after %d attempts", maxAttempts)
}

// GenerateUniqueInvoiceID generates a unique invoice ID in the format SL-YYYYMMDD-XXXX
func GenerateUniqueInvoiceID(db *gorm.DB) (string, error) {
	const prefix = "SL"
	const maxAttempts = 100 // Maximum attempts to generate a unique ID

	// Get the current date in YYYYMMDD format
	currentDate := time.Now().Format("20060102")

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Generate a random 4-digit number from UUID
		randomSuffix := uuid.New().ID() % 10000

		// Format the invoice ID
		invoiceID := fmt.Sprintf("%s-%s-%04d", prefix, currentDate, randomSuffix)

		// Check if the invoice ID exists in the database
		var count int64
		if err := db.Model(&models.Invoice{}).Where("invoice_id = ?", invoiceID).Count(&count).Error; err != nil {
			return "", fmt.Errorf("failed to check invoice ID uniqueness: %v", err)
		}

		// If the invoice ID is unique, return it
		if count == 0 {
			return invoiceID, nil
		}
	}

	return "", fmt.Errorf("failed to generate a unique invoice ID after %d attempts", maxAttempts)
}

// Contains checks if a slice contains a specific value.
func Contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// generateUniqueID creates a 4-digit unique identifier
func GenerateUniqueID() string {
	// Generate a random 4-digit number from UUID
	randomSuffix := uuid.New().ID() % 10000
	return fmt.Sprintf("%04d", randomSuffix)
}

// CapitalizeWords capitalizes the first letter of each word in a string
func CapitalizeWords(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
