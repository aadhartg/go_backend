// controllers/user_controller.go
package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UpdateUserRequest struct {
	FirstName string `json:"first_name" form:"first_name"`
	LastName  string `json:"last_name" form:"last_name"`
}

// UserProfile represents the user data to be sent in the response.
type UserProfile struct {
	ID           uint        `json:"id"`
	FirstName    string      `json:"first_name"`
	LastName     string      `json:"last_name"`
	Email        string      `json:"email"`
	Role         RoleProfile `json:"role"`
	ActiveStatus bool        `json:"active_status"`
	CreatedAt    time.Time   `json:"created_at"`
	NHI          string      `json:"nhi,omitempty"`
}

// RoleProfile represents the role data to be sent in the response.
type RoleProfile struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// GetUserProfileResponse represents the response structure.
type GetUserProfileResponse struct {
	Status  int         `json:"status"`
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    UserProfile `json:"data"`
}

// AdminUsersResponse represents the structured response for admin users list.
type AdminUsersResponse struct {
	Page         int           `json:"page"`
	PerPage      int           `json:"per_page"`
	Sort         string        `json:"sort"`
	SortColumn   string        `json:"sort_column"`
	SearchText   string        `json:"search_text"`
	Status       string        `json:"status"`
	TotalRecords int64         `json:"total_records"`
	TotalPages   int           `json:"total_pages"`
	Records      []UserProfile `json:"records"`
}

// CreateUserRequest represents the expected input for creating a user.
type CreateUserRequest struct {
	FirstName string `json:"first_name" validate:"required,max=50" form:"first_name"`
	LastName  string `json:"last_name" validate:"required,max=50" form:"last_name"`
	Email     string `json:"email" validate:"required,email,max=100" form:"email"`
}

type UpdateAdminUserRequest struct {
	FirstName    string `json:"first_name,omitempty" form:"first_name"`
	LastName     string `json:"last_name,omitempty" form:"last_name"`
	Email        string `json:"email,omitempty" form:"email"`
	Password     string `json:"password,omitempty" form:"password"`
	ActiveStatus bool   `json:"active_status,omitempty" form:"active_status"`
}

// Request structs for different update operations
type UpdateUserProfileRequest struct {
	FirstName string  `json:"first_name" form:"first_name"`
	LastName  *string `json:"last_name" form:"last_name"`
	Email     string  `json:"email" form:"email"`
}

type UpdateUserPasswordRequest struct {
	Password string `json:"password" form:"password"`
}

type UpdateUserStatusRequest struct {
	ActiveStatus bool `json:"active_status" form:"active_status"`
}

type UpdatePatientDetailsRequest struct {
	FirstName string `json:"first_name" form:"first_name"`
	LastName  string `json:"last_name" form:"last_name"`
	NHI       string `json:"nhi" form:"nhi"`

}

// UserProfileResponse represents the user data to be sent in the response for GetUserProfileHandler.
type UserProfileResponse struct {
	ID        uint        `json:"id"`
	FirstName string      `json:"first_name"`
	LastName  string      `json:"last_name"`
	Email     string      `json:"email"`
	Role      RoleProfile `json:"role"`
}

func UpdateUserInfoHandler(c *gin.Context) {
	// 1. This is a private route (enforced by AuthMiddleware)

	// Get the user from the context (set by AuthMiddleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotFound, nil)
		return
	}

	// Parse the request body
	var req UpdateUserRequest
	// Define allowed fields for this request
	allowedFields := []string{"first_name", "last_name"}

	// Use the common request parser for both JSON and form data, and validate allowed fields
	err := utils.ParseRequestBody(c.Request, &req, allowedFields)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Trim the first name
	req.FirstName = strings.TrimSpace(req.FirstName)

	// 5. Check if at least the first name is provided
	if req.FirstName == "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgAtleastPassSomeData, nil)
		return
	}

	// Validate first name
	if !utils.IsValidFirstName(req.FirstName) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgFirstNameValidation, nil)
		return
	}

	// Start a transaction
	tx := config.DB.Begin()

	// Update first name
	user.FirstName = req.FirstName

	// Handle last name
	if req.LastName != "" {
		req.LastName = strings.TrimSpace(req.LastName)
		if !utils.IsValidLastName(req.LastName) {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgLastNameValidation, nil)
			return
		}
		user.LastName = req.LastName
	} else {
		// Set last name to nil (or zero value)
		user.LastName = "" // Assuming the LastName field is a string
	}

	// 3. Update the user details
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

	// Prepare the response user profile
	userProfile := UserProfile{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Role: RoleProfile{
			ID:   user.Role.ID,
			Name: user.Role.Name,
		},
	}

	utils.JSONResponse(c, http.StatusOK, utils.MsgUserUpdateSuccessfully, userProfile)
}

// GetUserProfileHandler handles requests to fetch user profile details.
func GetUserProfileHandler(c *gin.Context) {
	// Get the user from the context (set by AuthMiddleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotFound, nil)
		return
	}

	// Prepare the user profile
	userProfile := UserProfileResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Role: RoleProfile{
			ID:   user.Role.ID,
			Name: user.Role.Name,
		},
	}

	// Send the encrypted response
	utils.JSONResponse(c, http.StatusOK, utils.MsgUserFetchedSuccessfully, userProfile)
}

// GetAdminUsersHandler handles requests to fetch the admin users list.
func GetAdminUsersHandler(c *gin.Context) {
	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "status"}

	// Parse query parameters with default values
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidQueryParameters, nil)
		return
	}

	// Default and validation for 'page'
	page := 1
	if val := query.Get("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPageParameter, nil)
			return
		}
	}

	// Default and validation for 'per_page'
	perPage := 10
	if val := query.Get("per_page"); val != "" {
		if pp, err := strconv.Atoi(val); err == nil && pp > 0 {
			perPage = pp
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPerPageParameter, nil)
			return
		}
	}

	// Default and validation for 'sort'
	sort := "desc"
	if val := strings.ToLower(query.Get("sort")); val == "asc" || val == "desc" {
		sort = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortParameter, nil)
		return
	}

	// Default and validation for 'sort_column'
	sortColumn := "created_at"
	validSortColumns := []string{"first_name", "last_name", "email", "active_status", "created_at"}
	if val := strings.ToLower(query.Get("sort_column")); val != "" {
		if utils.StringInSlice(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortColumnParameter, nil)
			return
		}
	}

	// Optional 'search_text'
	searchText := strings.TrimSpace(query.Get("search_text"))

	// Default and validation for 'status'
	status := "all"
	if val := strings.ToLower(query.Get("status")); val == "active" || val == "inactive" || val == "all" {
		status = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgStatusInvalid, nil)
		return
	}

	// Initialize GORM query with Debug for detailed logging
	db := config.DB.Debug().
		Model(&models.User{}).
		Joins("JOIN roles ON users.role_id = roles.id").
		Where("roles.name = ?", "admin").
		Where("users.is_deleted = ?", false)

	// Apply status filter
	if status == "active" {
		db = db.Where("users.active_status = ?", true)
	} else if status == "inactive" {
		db = db.Where("users.active_status = ?", false)
	}

	// Apply search filter
	if searchText != "" {
		searchPattern := "%" + searchText + "%"
		db = db.Where("roles.name = ?", "admin"). // Ensure only admin users are selected
								Where(
				"CONCAT(users.first_name, ' ', users.last_name) ILIKE ? OR users.email ILIKE ? OR CAST(users.created_at AS TEXT) ILIKE ?",
				searchPattern, searchPattern, searchPattern,
			)
	}

	// Get total records count
	var totalRecords int64
	if err := db.Count(&totalRecords).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCountRecords, nil)
		return
	}

	// Calculate total pages
	var totalPages int
	if totalRecords == 0 {
		totalPages = 0
	} else {
		totalPages = int((totalRecords + int64(perPage) - 1) / int64(perPage))
	}

	// Apply sorting
	db = db.Order(sortColumn + " " + sort)

	// Apply pagination
	offset := (page - 1) * perPage
	db = db.Limit(perPage).Offset(offset)

	// Fetch records
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Prepare user profiles
	var userProfiles []UserProfile
	for _, u := range users {
		profile := UserProfile{
			ID:        u.ID,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Email:     u.Email,
			Role: RoleProfile{
				ID:   u.Role.ID,
				Name: u.Role.Name,
			},
			ActiveStatus: u.ActiveStatus,
			CreatedAt:    u.CreatedAt,
		}
		userProfiles = append(userProfiles, profile)
	}

	// Prepare the response
	response := AdminUsersResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		Status:       status,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      userProfiles,
	}

	// Send the response
	utils.JSONResponse(c, http.StatusOK, utils.MsgUserListFetchedSuccessfully, response)
}

// CreateUserHandler handles the creation of a new admin user.
func CreateUserHandler(c *gin.Context) {
	// 1. Get the user from the context (set by AuthMiddleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Start transaction at the beginning
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Fetch the current user details
	var currentUser models.User
	if err := tx.Where("id = ? AND is_deleted = ?", user.ID, false).First(&currentUser).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// 1. Parse and Validate Request Body
	var req CreateUserRequest

	// Define allowed fields for this request
	allowedFields := []string{"email", "first_name", "last_name"}

	// Use the common request parser for both JSON and form data, and validate allowed fields
	err := utils.ParseRequestBody(c.Request, &req, allowedFields)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// 2. Validate Input Data
	if req.Email == "" || req.FirstName == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgFirstNameEmailRequired, nil)
		return
	}

	// Trim and validate first name
	req.FirstName = strings.TrimSpace(req.FirstName)
	if req.FirstName == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgAtleastPassSomeData, nil)
		return
	}

	if !utils.IsValidFirstName(req.FirstName) {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgFirstNameValidation, nil)
		return
	}

	// Validate last name
	req.LastName = strings.TrimSpace(req.LastName)
	if req.LastName != "" && !utils.IsValidLastName(req.LastName) {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgLastNameValidation, nil)
		return
	}

	// Validate email
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !utils.IsValidEmail(req.Email) {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgEmailValidation, nil)
		return
	}

	// 3. Check if the Email Already Exists
	var existingUser models.User
	if err := tx.Where("email = ? AND is_deleted = ?", req.Email, false).First(&existingUser).Error; err == nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusConflict, utils.MsgEmailAlreadyInUse, nil)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// 4. Retrieve Admin Role ID
	var adminRole models.Role
	if err := tx.Where("name = ? AND is_deleted = ?", "admin", false).First(&adminRole).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusNotFound, utils.MsgAdminRoleNotFound, nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// 5. Generate and Hash Password
	password := utils.GenerateSecurePassword()
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToSecurePassword, nil)
		return
	}

	// 6. Create the User
	newUser := models.User{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		HashPassword: hashedPassword,
		RoleID:       adminRole.ID,
		ActiveStatus: true,
		IsDeleted:    false,
	}

	if err := tx.Create(&newUser).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedCreateUser, nil)
		return
	}

	// 7. Send Email to the User
	appUrl := config.AppConfig.AppUrl
	emailBody := emails.WelcomeEmail(newUser.FirstName, newUser.LastName, newUser.Email, password, appUrl)
	if err := config.SendEmail([]string{newUser.Email}, "Welcome to Our Platform", emailBody); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedSentEmail, nil)
		return
	}

	// 8. Notify All Super Admin Users
	messageTemplate := fmt.Sprintf(
		"%s %s has successfully created a new account: %s %s.",
		currentUser.FirstName, currentUser.LastName, newUser.FirstName, newUser.LastName,
	)
	notificationUsername := fmt.Sprintf(
		"%s %s",
		currentUser.FirstName, currentUser.LastName,
	)
	roleNames := []string{"super-admin"}
	metadata := map[string]interface{}{
		"email":  newUser.Email,
		"module": "Manage Staff",
	}
	err = services.NotifyUsersByRoles(
		nil,
		tx,
		roleNames,
		"New Staff Member Account Created",
		"Admin Management",
		messageTemplate,
		notificationUsername,
		"users",
		newUser.ID,
		metadata,
	)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed To send Notification", nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedCreateUser, nil)
		return
	}

	utils.JSONResponse(c, http.StatusCreated, utils.MsgUserCreatedSuccessfully, nil)
}

// DeleteUserHandler marks the user as deleted (sets is_deleted to true) based on the user ID.
func DeleteUserHandler(c *gin.Context) {
	// Extract user ID from URL
	userID := c.Param("id")

	// Find the user based on ID and check if it's already deleted
	var existingUser models.User
	if err := config.DB.Where("id = ? AND is_deleted = ?", userID, false).First(&existingUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusNotFound, utils.MsgUserAlreadyDeleted, nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Mark user as deleted (set is_deleted to true)
	existingUser.IsDeleted = true

	// Create a transaction for updating the user
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Save(&existingUser).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToDeleteUser, nil)
		return
	}

	// 1. Get the user from the context (set by AuthMiddleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Fetch the full user details from the database
	var currentUser models.User
	if err := config.DB.Where("id = ? AND is_deleted = ?", user.ID, false).First(&currentUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Notify all super admins about the user deletion
	messageTemplate := "%s %s has deleted a account %s %s"
	notificationUsername := fmt.Sprintf(
		"%s %s",
		currentUser.FirstName, currentUser.LastName,
	)
	metadata := map[string]interface{}{
		"email":  currentUser.Email,
		"module": "Manage Staff",
	}
	err := services.NotifyUsersByRoles(
		config.DB,
		tx,                      // Use the current transaction
		[]string{"super-admin"}, // Notify only super admins
		"User Deleted",          // Title
		"Admin Management",
		fmt.Sprintf(messageTemplate, currentUser.FirstName, currentUser.LastName, existingUser.FirstName, existingUser.LastName), // Message
		notificationUsername,
		"users",         // Entity type
		existingUser.ID, // Entity ID
		metadata,
	)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToSendNotification, nil)
		return
	}
	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, utils.MsgUserDeleteSuccessfully, nil)
}

// Common function to fetch existing user
func getExistingUser(userID string) (*models.User, error) {
	var user models.User
	err := config.DB.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error
	return &user, err
}

// Helper functions for update operations
func updateUserProfile(user *models.User, req *UpdateUserProfileRequest) error {
	// Validate required fields
	if req.FirstName == "" || req.Email == "" {
		return errors.New("first name and email are required")
	}

	// Validate and update FirstName
	if !utils.IsValidFirstName(req.FirstName) {
		return errors.New(utils.MsgFirstNameValidation)
	}
	user.FirstName = req.FirstName

	// Validate and update LastName if provided
	if req.LastName != nil {
		if !utils.IsValidLastName(*req.LastName) {
			return errors.New(utils.MsgLastNameValidation)
		}
		user.LastName = *req.LastName
	}

	// Validate and update Email
	if !utils.IsValidEmail(req.Email) {
		return errors.New(utils.MsgEmailValidation)
	}

	// Check for existing email in the database
	var existingUser models.User
	if err := config.DB.Where("email = ? AND id != ?", req.Email, user.ID).First(&existingUser).Error; err == nil {
		return errors.New(utils.MsgEmailAlreadyInUse)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("database error while checking email uniqueness")
	}

	user.Email = req.Email
	return nil
}

func saveUserAndNotify(tx *gorm.DB, user *models.User, req *UpdateUserProfileRequest, changes []string) error {
	// Save user changes
	if err := tx.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Send email notification
	if len(changes) > 0 {
		emailBody := emails.UserDetailsUpdatedEmail(user.FirstName, user.LastName, user.Email, config.AppConfig.AppUrl)
		if err := config.SendEmail([]string{user.Email}, "Your Profile Details Updated", emailBody); err != nil {
			return fmt.Errorf("failed to send profile update notification: %w", err)
		}
	}

	return nil
}

// UpdateUserProfileHandler handles updating user's profile information
func UpdateUserProfileHandler(c *gin.Context) {
	// Start transaction
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
	}()

	// Get the current user from context
	currentUser, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Verify current user exists and is active
	if err := config.DB.Where("id = ? AND is_deleted = ?", currentUser.ID, false).First(&currentUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
		return
	}

	// Parse and validate request body
	var req UpdateUserProfileRequest
	if err := utils.ParseRequestBody(c.Request, &req, []string{"first_name", "last_name", "email"}); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Get target user to update
	userID := c.Param("id")
	existingUser, err := getExistingUser(userID)
	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, utils.MsgUserNotFound, nil)
		return
	}

	// Track changes for notifications
	changes := make([]string, 0)
	if req.FirstName != "" && req.FirstName != existingUser.FirstName {
		changes = append(changes, fmt.Sprintf("First Name changed from %s to %s", existingUser.FirstName, req.FirstName))
	}
	if req.LastName != nil && *req.LastName != existingUser.LastName {
		changes = append(changes, fmt.Sprintf("Last Name changed from %s to %s", existingUser.LastName, *req.LastName))
	}
	if req.Email != "" && req.Email != existingUser.Email {
		changes = append(changes, fmt.Sprintf("Email changed from %s to %s", existingUser.Email, req.Email))
	}

	// Update user profile
	if err := updateUserProfile(existingUser, &req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Save changes and send notifications
	if err := saveUserAndNotify(tx, existingUser, &req, changes); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// Only send notifications if there were changes
	if len(changes) > 0 {
		// Notify super admins
		notificationMessage := fmt.Sprintf(
			"%s %s updated the profile of %s %s. Changes: %s",
			currentUser.FirstName, currentUser.LastName, existingUser.FirstName, existingUser.LastName,
			strings.Join(changes, ". "),
		)
		notificationUsername := fmt.Sprintf(
			"%s %s",
			currentUser.FirstName, currentUser.LastName,
		)
		metadata := map[string]interface{}{
			"email":  currentUser.Email,
			"module": "Manage Staff",
		}
		if err := services.NotifyUsersByRoles(
			config.DB,
			tx,
			[]string{"super-admin"},
			"User Profile Updated",
			"Admin Management",
			notificationMessage,
			notificationUsername,
			"users",
			existingUser.ID,
			metadata,
		); err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToSendNotification, nil)
			return
		}

		// Notify the user
		userNotificationMessage := fmt.Sprintf("Your profile was updated. Changes: %s", strings.Join(changes, ". "))
		if err := services.CreateNotification(
			config.DB,
			existingUser.ID,
			"Profile Updated",
			userNotificationMessage,
			"Admin Management",
			notificationUsername,
			"users",
			existingUser.ID,
			metadata,
		); err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToSendNotification, nil)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, utils.MsgAdminUserUpdateSuccessfully, nil)
}

// Helper function for password updates
func updateUserPassword(user *models.User, password string) error {
	if !utils.IsValidPassword(password) {
		return errors.New(utils.MsgPasswordValidation)
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return errors.New(utils.MsgFailedToSecurePassword)
	}
	user.HashPassword = hashedPassword
	user.Token = ""

	return nil
}

func saveUserAndSendPasswordNotification(tx *gorm.DB, user *models.User, password string) error {
	// Save the updated user
	if err := tx.Save(user).Error; err != nil {
		return errors.New(utils.MsgFailedToUpdateAdminUser)
	}

	// Send an email to the user with their updated password
	emailBody := emails.PasswordUpdatedEmail(user.FirstName, user.LastName, user.Email, password, config.AppConfig.AppUrl)
	if err := config.SendEmail([]string{user.Email}, "Your password has been updated", emailBody); err != nil {
		return errors.New("Failed to send password update email")
	}

	return nil
}

// UpdateUserPasswordHandler handles updating a user's password.
func UpdateUserPasswordHandler(c *gin.Context) {
	// 1. Get the user from the context (set by AuthMiddleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Start transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToStartTransaction, nil)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Fetch the current user details
	var currentUser models.User
	if err := tx.Where("id = ? AND is_deleted = ?", user.ID, false).First(&currentUser).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Parse the request body
	var req UpdateUserPasswordRequest
	if err := utils.ParseRequestBody(c.Request, &req, []string{"password"}); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Fetch the existing user whose password is being updated
	existingUser, err := getExistingUser(c.Param("id"))
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusNotFound, utils.MsgUserNotFound, nil)
		return
	}

	// Validate and update the password
	if err := updateUserPassword(existingUser, req.Password); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Save the user and notify them via email
	if err := saveUserAndSendPasswordNotification(tx, existingUser, req.Password); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	notificationUsername := fmt.Sprintf(
		"%s %s",
		currentUser.FirstName, currentUser.LastName,
	)
	// Create notification for the user
	userNotificationMessage := "Your password was updated. If you didn't initiate this change, please contact support immediately."
	metadata := map[string]interface{}{
		"email":  currentUser.Email,
		"module": "Manage Staff",
	}
	err = services.CreateNotification(
		tx, // Use transaction instead of config.DB
		existingUser.ID,
		"Password Updated",
		userNotificationMessage,
		"Admin Management",
		notificationUsername,
		"users",
		existingUser.ID,
		metadata,
	)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToSendNotificationUser, nil)
		return
	}

	// Notify all super admins
	adminNotificationMessage := fmt.Sprintf(
		"%s %s updated the password for %s %s.",
		currentUser.FirstName, currentUser.LastName, existingUser.FirstName, existingUser.LastName,
	)

	err = services.NotifyUsersByRoles(
		nil, // Use transaction instead of config.DB
		tx,
		[]string{"super-admin"},
		"Admin Password Updated",
		"Admin Management",
		adminNotificationMessage,
		notificationUsername,
		"users",
		existingUser.ID,
		metadata,
	)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToNotifySuperAdmins, nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCommitTransaction, nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, utils.MsgPasswordUpdatedSuccessfully, nil)
}

func updateUserStatus(tx *gorm.DB, currentUser *models.User, targetUser *models.User, newStatus bool) error {
	if targetUser.ActiveStatus == newStatus {
		return errors.New("status is already set to the requested value")
	}

	targetUser.ActiveStatus = newStatus
	if err := tx.Save(targetUser).Error; err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	// Determine status message
	statusMessage := "activated"
	if !newStatus {
		statusMessage = "deactivated"
	}

	// Send email notification to the target user
	emailBody := emails.UserStatusChangedEmail(
		targetUser.FirstName,
		targetUser.LastName,
		statusMessage,
		config.AppConfig.AppUrl,
	)
	if err := config.SendEmail(
		[]string{targetUser.Email},
		"Your Account Status has Changed",
		emailBody,
	); err != nil {
		return fmt.Errorf("failed to send status change notification: %w", err)
	}

	notificationUsername := fmt.Sprintf(
		"%s %s",
		currentUser.FirstName, currentUser.LastName,
	)
	metadata := map[string]interface{}{
		"email":  currentUser.Email,
		"module": "Manage Staff",
	}

	// Create notification for the user
	userNotificationMessage := fmt.Sprintf("Your account has been %s", statusMessage)
	if err := services.CreateNotification(
		tx,
		targetUser.ID,
		"Account Status Updated",
		userNotificationMessage,
		"Admin Management",
		notificationUsername,
		"users",
		targetUser.ID,
		metadata,
	); err != nil {
		return fmt.Errorf("failed to create user notification: %w", err)
	}

	// Notify all super admins
	superAdminNotificationMessage := fmt.Sprintf(
		"%s %s has %s the account of %s %s",
		currentUser.FirstName,
		currentUser.LastName,
		statusMessage,
		targetUser.FirstName,
		targetUser.LastName,
	)

	if err := services.NotifyUsersByRoles(
		nil,
		tx,
		[]string{"super-admin"},
		"User Status Updated",
		"Admin Management",
		superAdminNotificationMessage,
		notificationUsername,
		"users",
		targetUser.ID,
		metadata,
	); err != nil {
		return fmt.Errorf("failed to notify super admins: %w", err)
	}

	return nil
}

// UpdateUserStatusHandler handles user status updates
func UpdateUserStatusHandler(c *gin.Context) {
	// Start transaction
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
	}()

	// Get current user from context
	currentUser, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Parse request
	var req UpdateUserStatusRequest
	if err := utils.ParseRequestBody(c.Request, &req, []string{"active_status"}); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Get target user
	targetUser, err := getExistingUser(c.Param("id"))
	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, utils.MsgUserNotFound, nil)
		return
	}

	// Update status and send notifications
	if err := updateUserStatus(tx, currentUser, targetUser, req.ActiveStatus); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, utils.MsgAdminUserUpdateSuccessfully, nil)
}

// GetAllPatientHandler handles requests to fetch the admin users list.
func GetAllPatientHandler(c *gin.Context) {
	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "status" ,"role"}

	// Parse query parameters with default values
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidQueryParameters, nil)
		return
	}

	// Default and validation for 'page'
	page := 1
	if val := query.Get("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPageParameter, nil)
			return
		}
	}

	// Default and validation for 'per_page'
	perPage := 10
	if val := query.Get("per_page"); val != "" {
		if pp, err := strconv.Atoi(val); err == nil && pp > 0 {
			perPage = pp
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPerPageParameter, nil)
			return
		}
	}

	// Default and validation for 'sort'
	sort := "desc"
	if val := strings.ToLower(query.Get("sort")); val == "asc" || val == "desc" {
		sort = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortParameter, nil)
		return
	}

	// Default and validation for 'sort_column'
	sortColumn := "created_at"
	validSortColumns := []string{"first_name", "last_name", "email", "active_status", "created_at", "nhi"}
	if val := strings.ToLower(query.Get("sort_column")); val != "" {
		if utils.StringInSlice(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortColumnParameter, nil)
			return
		}
	}

	// Optional 'search_text'
	searchText := strings.TrimSpace(query.Get("search_text"))

	// Default and validation for 'status'
	status := "all"
	if val := strings.ToLower(query.Get("status")); val == "active" || val == "inactive" || val == "all" {
		status = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgStatusInvalid, nil)
		return
	}

	// Default role for 'status'
	role := "patient"
	if val := strings.ToLower(query.Get("role")); val == "patient" || val == "coordinator" {
		role = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgStatusInvalid, nil)
		return
	}


	// Initialize GORM query with Debug for detailed logging
	db := config.DB.Debug().
		Model(&models.User{}).
		Joins("JOIN roles ON users.role_id = roles.id").
		//Where("roles.name = ?", "patient").
		Where("roles.name = ?", role).
		Where("users.is_deleted = ?", false)

	// Apply status filter
	if status == "active" {
		db = db.Where("users.active_status = ?", true)
	} else if status == "inactive" {
		db = db.Where("users.active_status = ?", false)
	}

	if searchText != "" {
		// Convert search text to lowercase for case-insensitive search
		searchText = strings.ToLower(searchText)
		searchPattern := "%" + searchText + "%"

		// Parse search text for date patterns using the utility function
		dateResult := utils.ParseSearchTextForDate(searchText)

		// Check if this is a date search
		if dateResult.IsDate {
			// Date search: search for records created on the specific date(s)
			var dateConditions []string
			var dateArgs []interface{}

			// Handle single date vs multiple dates
			if dateResult.Date != nil {
				// Single date
				startOfDay := time.Date(dateResult.Date.Year(), dateResult.Date.Month(), dateResult.Date.Day(), 0, 0, 0, 0, dateResult.Date.Location())
				endOfDay := startOfDay.Add(24 * time.Hour)
				dateConditions = append(dateConditions, "(users.created_at >= ? AND users.created_at < ?)")
				dateArgs = append(dateArgs, startOfDay, endOfDay)
			} else if len(dateResult.Dates) > 0 {
				// Multiple dates - create OR conditions for each date
				for _, date := range dateResult.Dates {
					startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
					endOfDay := startOfDay.Add(24 * time.Hour)
					dateConditions = append(dateConditions, "(users.created_at >= ? AND users.created_at < ?)")
					dateArgs = append(dateArgs, startOfDay, endOfDay)
				}
			}

			// Build the combined search query
			combinedSearch := `
				(LOWER(CONCAT(users.first_name, ' ', users.last_name)) LIKE ? OR 
				LOWER(users.email) LIKE ?`

			// Add date conditions if any
			if len(dateConditions) > 0 {
				combinedSearch += " OR " + strings.Join(dateConditions, " OR ")
			}
			combinedSearch += ")"

			// Combine text args with date args
			allArgs := append([]interface{}{searchPattern, searchPattern}, dateArgs...)
			db = db.Where(combinedSearch, allArgs...)
		} else if dateResult.IsDayOnly {
			// Day-only search: include day-of-month filter
			combinedSearch := `
				(LOWER(CONCAT(users.first_name, ' ', users.last_name)) LIKE ? OR 
				LOWER(users.email) LIKE ? OR
				EXTRACT(DAY FROM users.created_at) = ?)
			`
			db = db.Where(combinedSearch, searchPattern, searchPattern, *dateResult.Day)
		} else {
			// Text search only
			combinedSearch := `
				LOWER(CONCAT(users.first_name, ' ', users.last_name)) LIKE ? OR 
				LOWER(users.email) LIKE ?
			`
			db = db.Where(combinedSearch, searchPattern, searchPattern)
		}
	}

	// Get total records count
	var totalRecords int64
	if err := db.Count(&totalRecords).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCountRecords, nil)
		return
	}

	// Calculate total pages
	var totalPages int
	if totalRecords == 0 {
		totalPages = 0
	} else {
		totalPages = int((totalRecords + int64(perPage) - 1) / int64(perPage))
	}

	// Apply sorting
	db = db.Order(sortColumn + " " + sort)

	// Apply pagination
	offset := (page - 1) * perPage
	db = db.Limit(perPage).Offset(offset)

	// Fetch records
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Prepare user profiles
	var userProfiles []UserProfile
	for _, u := range users {
		profile := UserProfile{
			ID:        u.ID,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Email:     u.Email,
			Role: RoleProfile{
				ID:   u.Role.ID,
				Name: u.Role.Name,
			},
			ActiveStatus: u.ActiveStatus,
			CreatedAt:    u.CreatedAt,
			NHI:      u.NHI,
		}
		userProfiles = append(userProfiles, profile)
	}

	// Prepare the response
	response := AdminUsersResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		Status:       status,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      userProfiles,
	}

	// Send the response
	utils.JSONResponse(c, http.StatusOK, utils.MsgUserListFetchedSuccessfully, response)
}


// UpdatePatientDetailsHandler handles updating patient details (first name, last name, NHI)
func UpdatePatientDetailsHandler(c *gin.Context) {
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
	}()
	patientID := c.Param("id")
	if patientID == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Patient ID is required", nil)
		return
	}
	var req UpdatePatientDetailsRequest
	allowedFields := []string{"first_name", "last_name", "nhi", "firstName", "lastName"}
	if err := utils.ParseRequestBody(c.Request, &req, allowedFields); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}


	if req.FirstName == "" && req.LastName == "" && req.NHI == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgAtleastPassSomeData, nil)
		return
	}
	var patient models.User
	if err := config.DB.Where("id = ? AND is_deleted = ?", patientID, false).First(&patient).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusNotFound, "Patient not found", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
		return
	}
	var role models.Role
	if err := config.DB.Where("id = ?", patient.RoleID).First(&role).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch user role", nil)
		return
	}
	if role.Name != "patient" && role.Name != "coordinator" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "This endpoint is only for patients", nil)
		return
	}
	changes := []string{}
	if req.FirstName != "" {
		req.FirstName = strings.TrimSpace(req.FirstName)
		if !utils.IsValidFirstName(req.FirstName) {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgFirstNameValidation, nil)
			return
		}
		if req.FirstName != patient.FirstName {
			changes = append(changes, fmt.Sprintf("First name changed from %s to %s", patient.FirstName, req.FirstName))
			patient.FirstName = req.FirstName
		}
	}
	fmt.Println("--- req: ", req)

	if req.LastName != "" {
		req.LastName = strings.TrimSpace(req.LastName)
		if !utils.IsValidLastName(req.LastName) {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgLastNameValidation, nil)
			return
		}
		fmt.Println("--- last prev: ", patient.LastName)
		fmt.Println("--- last new: ", req.LastName)
		if req.LastName != patient.LastName {
			changes = append(changes, fmt.Sprintf("Last name changed from %s to %s", patient.LastName, req.LastName))
			patient.LastName = req.LastName
		}
	}
	if req.NHI != "" {
		req.NHI = strings.TrimSpace(req.NHI)
		if len(req.NHI) > 20 {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "NHI number must be 20 characters or less", nil)
			return
		}

		var existingPatient models.User
		err := tx.Where(
			"nhi = ? AND id != ? AND is_deleted = ?",
			req.NHI,
			patient.ID,
			false,
		).First(&existingPatient).Error

		if err == nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "NHI number already exists", nil)
			return
		}

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
			return
		}

		if req.NHI != patient.NHI {
			changes = append(changes, fmt.Sprintf("NHI number changed from %s to %s", patient.NHI, req.NHI))
			patient.NHI = req.NHI
		}
	}
	if len(changes) == 0 {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "No changes were made", nil)
		return
	}
	if err := tx.Save(&patient).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update patient details", nil)
		return
	}
	currentUser, ok := middlewares.GetUserFromContext(c)
	if ok {
		userNotificationMessage := fmt.Sprintf("Your profile was updated. Changes: %s", strings.Join(changes, ". "))
		notificationUsername := fmt.Sprintf("%s %s", currentUser.FirstName, currentUser.LastName)
		metadata := map[string]interface{}{
			"email":  currentUser.Email,
			"module": "Patient Management",
		}
		if err := services.CreateNotification(config.DB, patient.ID, "Profile Updated", userNotificationMessage, "Patient Management", notificationUsername, "users", patient.ID, metadata); err != nil {
			fmt.Printf("Failed to create notification: %v\n", err)
		}
		superAdminMessage := fmt.Sprintf("%s %s updated patient %s %s. Changes: %s", currentUser.FirstName, currentUser.LastName, patient.FirstName, patient.LastName, strings.Join(changes, ". "))
		if err := services.NotifyUsersByRoles(config.DB, tx, []string{"super-admin"}, "Patient Details Updated", "Patient Management", superAdminMessage, notificationUsername, "users", patient.ID, metadata); err != nil {
			fmt.Printf("Failed to notify super admins: %v\n", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}
	utils.JSONResponse(c, http.StatusOK, "Patient details updated successfully", nil)
}
