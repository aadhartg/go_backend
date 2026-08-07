// controllers/patient_registration_controller.go

package controllers

import (
	"net/http"
	"strings"
	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"

	"log"

	"github.com/gin-gonic/gin"
)

type RegisterRequest struct {
	FirstName string `json:"firstName" form:"firstName" binding:"required"`
	LastName  string `json:"lastName" form:"lastName" binding:"required"`
	Email     string `json:"email" form:"email" binding:"required"`
	Role      string `json:"role" form:"role" binding:"required"`
}

func PatientRegister(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBind(&req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}
	// Convert email to lowercase
	req.Email = strings.ToLower(req.Email)

	tx := config.DB.Begin()

	// Validate patient data via service
	if err := services.ValidatePatientRequestData(req.FirstName, req.LastName, req.Email); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Insert patient record
	user, err := services.InsertPatientRegistration(tx, req.FirstName, req.LastName, req.Email, req.Role)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	user_role := req.Role

	//Generate new password
	newPassword := utils.GenerateSecurePassword()

	// Send the email
	emailBody := emails.WelcomeEmail(user.FirstName, user.LastName, req.Email, newPassword, config.AppConfig.AppUrl)
	if err := config.SendEmail([]string{req.Email}, "Welcome to Our Platform", emailBody); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedSentEmail, nil)
		return
	}

	// Hash and update the new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Update the user struct
	user.HashPassword = hashedPassword
	user.Token = ""

	// Save the updated user to the database
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Notify admin and super-admin about new patient signup
	notificationTitle := "New Patient Signup"
	notificationMessage := "A new " + user_role + " has signed up: " + user.FirstName + " " + user.LastName + " (" + user.Email + ")"
	notificationUsername := user.FirstName + " " + user.LastName
	err = services.NotifyUsersByRoles(
		config.DB,
		tx, // No active transaction, but function will fallback to DB
		[]string{"admin", "super-admin"},
		notificationTitle,
		"Patient", // notificationType
		notificationMessage,
		notificationUsername,
		"user",  // entityType
		user.ID, // entityID
		nil,     // metadata
	)
	if err != nil {
		// Log the error but don't fail the registration
		log.Printf("Failed to notify admins about new "+user_role+" signup: %v", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	msg := strings.ToUpper(req.Role[:1]) + req.Role[1:]
	utils.JSONResponse(c, http.StatusOK, msg+" registered successfully", user)

}
