// controllers/forget_password.go
package controllers

import (
	"net/http"
	"strings"

	"gorm.io/gorm"

	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

// PasswordForgetRequest represents the expected request body
type PasswordForgetRequest struct {
	Email string `json:"email" form:"email"`
}

// ForgetPasswordHandler handles password reset requests.
func ForgetPasswordHandler(c *gin.Context) {
	var req PasswordForgetRequest
	// Define allowed fields for this request
	allowedFields := []string{"email"}

	// Use the common request parser for both JSON and form data, and validate allowed fields
	err := utils.ParseRequestBody(c.Request, &req, allowedFields)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate input
	if req.Email == "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgMissingEmail, nil)
		return
	}

	// Trim spaces and convert email to lowercase
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Validate email format
	if !utils.IsValidEmail(req.Email) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidEmailFormat, nil)
		return
	}

	// Find the user by email using the common function
	user, err := utils.FindUserByEmail(config.DB, req.Email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgEmailNotExist, nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Generate a secure random password
	newPassword := utils.GenerateSecurePassword()

	// Start a transaction
	tx := config.DB.Begin()

	// Defer a rollback in case of failure
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Send the email
	emailBody := emails.ResetPasswordEmail(user.FirstName, user.LastName, req.Email, newPassword, config.AppConfig.AppUrl)
	if err := config.SendEmail([]string{req.Email}, "Password Reset", emailBody); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedSentEmail, nil)
		return
	}

	// Hash and update the new password in the database
	hashedPassword, err := utils.HashPassword(newPassword) // Implement HashPassword function
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Update user's hashed password and remove the token
	user.HashPassword = hashedPassword
	user.Token = ""
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Send the response
	utils.JSONResponse(c, http.StatusOK, utils.MsgForgetSuccessfully, nil)
}
