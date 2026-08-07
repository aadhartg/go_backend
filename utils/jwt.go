package utils

import (
	"errors"
	"strconv"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/models"

	"github.com/golang-jwt/jwt/v4"
)

// GenerateJWT generates a JWT token for the given user and encrypts the user ID.
func GenerateJWT(user models.User) (string, error) {
	config.LoadEnv() // Ensure environment variables are loaded
	cfg := config.AppConfig
	jwtSecret := []byte(cfg.JWTSecret) // Access AppConfig directly

	// Convert user.ID (uint) to string before encryption
	userIDStr := strconv.FormatUint(uint64(user.ID), 10)

	// Encrypt the user ID
	encryptedUserID, err := Encrypt(userIDStr)
	if err != nil {
		return "", err
	}

	// Define token claims, using the encrypted user ID
	claims := jwt.MapClaims{
		"id":  encryptedUserID,
		"exp": time.Now().Add(time.Hour * 24).Unix(), // Token expires after 24 hours
		"iat": time.Now().Unix(),
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT decrypts the token, validates it, and decrypts the user ID.
func ValidateJWT(tokenString string) (jwt.MapClaims, error) {
	cfg := config.AppConfig
	jwtSecret := []byte(cfg.JWTSecret)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Decrypt the user ID
		if encryptedID, ok := claims["id"].(string); ok {
			decryptedID, err := Decrypt(encryptedID)
			if err != nil {
				return nil, errors.New("invalid user ID encryption")
			}

			// Convert the decrypted user ID string back to uint
			userIDUint, err := strconv.ParseUint(decryptedID, 10, 64)
			if err != nil {
				return nil, errors.New("invalid user ID format")
			}
			claims["id"] = uint(userIDUint) // Convert back to uint and store in claims
		} else {
			return nil, errors.New("invalid token claims")
		}

		return claims, nil
	} else {
		return nil, jwt.ErrSignatureInvalid
	}
}
