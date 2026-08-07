// utils/password.go
package utils

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes the given password using bcrypt.
// It returns the hashed password as a string and any error encountered.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

const (
	letters        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers        = "0123456789"
	specialChars   = "@"
	passwordLength = 12
)

// GenerateSecurePassword generates a random password with specified constraints
func GenerateSecurePassword() string {
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	password := make([]byte, passwordLength)

	// Ensure at least one lowercase letter
	password[0] = letters[rand.Intn(26)] // Lowercase
	// Ensure at least one uppercase letter
	password[1] = letters[26+rand.Intn(26)] // Uppercase
	// Ensure at least one number
	password[2] = numbers[rand.Intn(len(numbers))]
	// Ensure at least one special character
	password[3] = specialChars[0] // Only "@" is allowed

	for i := 4; i < passwordLength; i++ {
		allChars := letters + numbers + specialChars
		password[i] = allChars[rand.Intn(len(allChars))]
	}

	// Shuffle the password to ensure randomness
	rand.Shuffle(len(password), func(i, j int) {
		password[i], password[j] = password[j], password[i]
	})

	return string(password)
}
