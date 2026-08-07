package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

type EncryptedRequest struct {
	Data json.RawMessage `json:"data" validate:"required"`
}

// EncryptionDetails handles the encryption of data
func EncryptionDetails(c *gin.Context) {
	// Parse the request body
	var req EncryptedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	// Log the raw data (optional, for debugging)
	log.Println("Received data:", string(req.Data))

	// Encrypt the JSON string
	encryptedData, err := utils.Encrypt(string(req.Data))
	if err != nil {
		log.Printf("Encryption error: %v", err)
		utils.JSONResponse(c, http.StatusInternalServerError, "Error encrypting data", map[string]string{"error": err.Error()})
		return
	}

	// Return the encrypted data
	utils.JSONResponse(c, http.StatusOK, "Success", encryptedData)
}
