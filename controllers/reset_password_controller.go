package controllers

import (
	"net/http"
	"strings"

	"theransticslabs/m/config"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type ResetPasswordRequest struct {
	OldPassword string `json:"old_password" form:"old_password"`
	NewPassword string `json:"new_password" form:"new_password"`
}

func ResetPasswordHandler(c *gin.Context) {
	// 1. Verify the token (this is done by the AuthMiddleware)

	// 2. Get the user from the context (set by AuthMiddleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotFound, nil)
		return
	}

	// 3. Parse the request body
	var req ResetPasswordRequest
	// Define allowed fields for this request
	allowedFields := []string{"old_password", "new_password"}

	// Use the common request parser for both JSON and form data, and validate allowed fields
	err := utils.ParseRequestBody(c.Request, &req, allowedFields)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Trim whitespace
	req.NewPassword = strings.TrimSpace(req.NewPassword)
	req.OldPassword = strings.TrimSpace(req.OldPassword)

	// Validate input
	if req.NewPassword == "" || req.OldPassword == "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgOldNewPasswordRequired, nil)
		return
	}

	// 4. Validate the new password
	if !utils.IsValidPassword(req.NewPassword) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPasswordFormat, nil)
		return
	}

	// 5. Check if old password and new password are the same
	if req.OldPassword == req.NewPassword {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgPasswordSame, nil)
		return
	}

	// 6. Verify the old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashPassword), []byte(req.OldPassword)); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidOldPassword, nil)
		return
	}

	// 7. Hash the new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToHashNewPassword, nil)
		return
	}

	// Update user's hashed password and remove the token
	user.HashPassword = hashedPassword
	user.Token = ""
	if err := config.DB.Save(user).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToUpdateNewPassword, nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, utils.MsgResetPasswordSuccessfully, nil)
}
