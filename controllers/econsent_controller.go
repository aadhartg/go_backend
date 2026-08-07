package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"

	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateEConsentFormRequest represents the request body for creating an e-consent form
type CreateEConsentFormRequest struct {
	ResearchID         uint      `json:"research_id" binding:"required"`
	Title              string    `json:"title" binding:"required,min=3,max=100"`
	VideoID            uint      `json:"video_id" binding:"required"`
	VideoDescription   string    `json:"video_description" binding:"required,min=10,max=500"`
	PersonalInfoCheck  bool      `json:"personal_info_check"`
	CoordinatorIDs     []uint    `json:"coordinator_ids" binding:"required,min=1"`
	PatientAssigned    []uint    `json:"patient_assigned" binding:"required,min=1"`
	ExpiryTime         time.Time `json:"expiry_time" binding:"required"`
	DeclarationFormIDs []uint    `json:"declaration_form_ids" binding:"required,min=1"`
}

// EditEConsentFormRequest represents the request body for editing an e-consent form
type EditEConsentFormRequest struct {
	Title                 string    `json:"title" binding:"required,min=3,max=100"`
	ResearchID            uint      `json:"research_id" binding:"required"`
	VideoID               uint      `json:"video_id" binding:"required"`
	VideoDescription      string    `json:"video_description" binding:"required,min=10,max=500"`
	PersonalInfoCheck     bool      `json:"personal_info_check"`
	CoordinatorIDs        []uint    `json:"coordinator_ids" binding:"required,min=1"`
	PatientAssigned       []uint    `json:"patient_assigned" binding:"required,min=1"`
	ExpiryTime            time.Time `json:"expiry_time" binding:"required"`
	DeclarationFormIDs    []uint    `json:"declaration_form_ids,omitempty"`
	IsOnlyPatientAddition bool      `json:"is_only_patient_addition,omitempty"`
}

// UserInfoResponse represents a user with ID and label
type UserInfoResponse struct {
	ID    uint   `json:"id"`
	Label string `json:"label"`
}

// EConsentFormResponse represents the response for e-consent form
type EConsentFormResponse struct {
	ID                uint                         `json:"id"`
	ResearchID        uint                         `json:"research_id"`
	Research          ResearchResponseForConsent   `json:"research"`
	Title             string                       `json:"title"`
	VideoID           uint                         `json:"video_id"`
	VideoDescription  string                       `json:"video_description"`
	PersonalInfoCheck bool                         `json:"personal_info_check"`
	Coordinators      []UserInfoResponse           `json:"coordinators"`
	PatientAssigned   []UserInfoResponse           `json:"patient_assigned"`
	ExpiryTime        string                       `json:"expiry_time"`
	Video             VideoResponseForConsent      `json:"video"`
	IsPublished       bool                         `json:"is_published"`
	CreatedAt         time.Time                    `json:"created_at"`
	UpdatedAt         time.Time                    `json:"updated_at"`
	Declarations      []ConsentDeclarationResponse `json:"declarations,omitempty"`
	Assignments       []EConsentAssignmentResponse `json:"assignments,omitempty"`
	PatientConsents   []PatientEConsentResponse    `json:"patient_consents,omitempty"`
	DeclarationForms  []map[string]interface{}     `json:"declaration_forms,omitempty"`
	Submitted 				bool										     `json:"submitted"`
	VersionNumber 		int 												 `json:"version_number"`
	Expired 					bool 												 `json:"expired"`
}

// ResearchResponse represents research data in the response
type ResearchResponseForConsent struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CategoryID  uint   `json:"category_id"`
	IsPublished bool   `json:"is_published"`
	UserName    string `json:"user_name"`
}

// VideoResponse represents video data in the response
type VideoResponseForConsent struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
}

// ConsentDeclarationResponse represents a declaration in the response (from Declaration table)
type ConsentDeclarationResponse struct {
	ID    uint   `json:"id"`
	Label string `json:"label"`
}

// EConsentAssignmentResponse represents an e-consent assignment in the response
type EConsentAssignmentResponse struct {
	ID         uint      `json:"id"`
	UserID     uint      `json:"user_id"`
	AssignedAt time.Time `json:"assigned_at"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// PatientEConsentResponse represents a patient e-consent in the response
type PatientEConsentResponse struct {
	ID            uint      `json:"id"`
	PatientID     uint      `json:"patient_id"`
	ConsentStatus string    `json:"consent_status"`
	ConsentedAt   time.Time `json:"consented_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// EConsentFormListResponse represents the structured response for e-consent forms list
type EConsentFormListResponse struct {
	Page         int                    `json:"page"`
	PerPage      int                    `json:"per_page"`
	Sort         string                 `json:"sort"`
	SortColumn   string                 `json:"sort_column"`
	SearchText   string                 `json:"search_text"`
	TotalRecords int64                  `json:"total_records"`
	TotalPages   int                    `json:"total_pages"`
	Records      []EConsentFormResponse `json:"records"`
}

// getUsersInfo fetches user information for a slice of user IDs
func getUsersInfo(userIDs []uint) ([]UserInfoResponse, error) {
	if len(userIDs) == 0 {
		return []UserInfoResponse{}, nil
	}

	var users []models.User
	if err := config.DB.Where("id IN (?)", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}

	userInfoMap := make(map[uint]string)
	for _, user := range users {
		userInfoMap[user.ID] = user.FirstName + " " + user.LastName
	}
	result := make([]UserInfoResponse, 0, len(userIDs))
	for _, id := range userIDs {
		if label, ok := userInfoMap[id]; ok {
			encryptedLabel, err := utils.Encrypt(label)
			if err != nil {
				return nil, err
			}
			result = append(result, UserInfoResponse{ID: id, Label: encryptedLabel})
		}
	}

	return result, nil
}

// extractPatientIDs is a utility function to extract patient IDs from various formats
func extractPatientIDs(input interface{}) ([]uint, error) {
	var patientIDs []uint
	switch v := input.(type) {
	case nil:
		return []uint{}, nil
	case string:
		if v == "" || v == "null" {
			return []uint{}, nil
		}
		if err := json.Unmarshal([]byte(v), &patientIDs); err != nil {
			return nil, err
		}
	case []uint:
		patientIDs = v
	case models.PatientAssignedArray:
		patientIDs = []uint(v)
	case *[]uint:
		patientIDs = *v
	case []interface{}:
		for _, id := range v {
			switch idVal := id.(type) {
			case float64:
				patientIDs = append(patientIDs, uint(idVal))
			case int:
				patientIDs = append(patientIDs, uint(idVal))
			}
		}
	default:
		return nil, fmt.Errorf("invalid patient IDs format: %T", input)
	}
	return patientIDs, nil
}

// CreateEConsentForm creates a new e-consent form
func CreateEConsentForm(c *gin.Context) {
	// Get user from context using the middleware helper``
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can create e-consent forms", nil)
		return
	}

	// Bind request data
	var req CreateEConsentFormRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate expiry time is in the future
	if req.ExpiryTime.Before(time.Now()) {
		utils.JSONResponse(c, http.StatusBadRequest, "Expiry time must be in the future", nil)
		return
	}

	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Verify that the research exists and is not deleted
	var research models.Research
	if err := tx.Where("id = ? AND is_deleted = ?", req.ResearchID, false).First(&research).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid research ID", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to validate research", nil)
		}
		return
	}
	// Only published research can be attached to e-consent
	if !research.IsPublished {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Only published research can be attached to e-consent", nil)
		return
	}

	// Verify that the video exists and is not deleted
	var video models.Video
	if err := tx.Where("id = ? AND is_deleted = ?", req.VideoID, false).First(&video).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid video ID", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to validate video", nil)
		}
		return
	}

	// Verify that all coordinators exist and are not deleted
	for _, coordinatorID := range req.CoordinatorIDs {
		var coordinator models.User
		if err := tx.Where("id = ? AND is_deleted = ?", coordinatorID, false).First(&coordinator).Error; err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid coordinator ID: "+strconv.FormatUint(uint64(coordinatorID), 10), nil)
			} else {
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to validate coordinator", nil)
			}
			return
		}
	}

	// Verify that all assigned patients exist, are not deleted, are patients (role_id = 4), and are active (active_status = true)
	var inactivePatientIDs []uint
	for _, patientID := range req.PatientAssigned {
		var patient models.User
		if err := tx.Where("id = ? AND is_deleted = ?", patientID, false).First(&patient).Error; err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid patient ID: "+strconv.FormatUint(uint64(patientID), 10), nil)
			} else {
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to validate patient", nil)
			}
			return
		}
		if patient.RoleID != 4 {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "Assigned user is not a patient", nil)
			return
		}
		if !patient.ActiveStatus {
			inactivePatientIDs = append(inactivePatientIDs, patientID)
		}
	}
	if len(inactivePatientIDs) > 0 {
		// Fetch names for better error message
		var inactivePatients []models.User
		if err := tx.Where("id IN (?)", inactivePatientIDs).Find(&inactivePatients).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "Some assigned patients are inactive", nil)
			return
		}
		var names []string
		for _, p := range inactivePatients {
			names = append(names, fmt.Sprintf("%s %s (ID: %d)", p.FirstName, p.LastName, p.ID))
		}
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "The following patients are inactive and cannot be assigned: "+strings.Join(names, ", "), nil)
		return
	}
	for _, patientID := range req.PatientAssigned {
		var patient models.User
		if err := tx.Where("id = ? AND is_deleted = ?", patientID, false).First(&patient).Error; err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid patient ID: "+strconv.FormatUint(uint64(patientID), 10), nil)
			} else {
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to validate patient", nil)
			}
			return
		}
		if patient.RoleID != 4 {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "Assigned user is not a patient", nil)
			return
		}
	}

	// Validate all declaration forms exist
	var declarationForms []models.DeclarationForm
	if req.DeclarationFormIDs != nil {
		if err := tx.Preload("Declarations").Where("id IN (?)", req.DeclarationFormIDs).Find(&declarationForms).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
			return
		}
		if len(declarationForms) != len(req.DeclarationFormIDs) {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
			return
		}
	}

	// Create e-consent form
	eConsentForm := models.EConsentForm{
		ResearchID:        req.ResearchID,
		CreatedBy:         user.ID,
		Title:             req.Title,
		VideoID:           req.VideoID,
		VideoDescription:  req.VideoDescription,
		PersonalInfoCheck: req.PersonalInfoCheck,
		PatientAssigned:   req.PatientAssigned,
		ExpiryTime:        req.ExpiryTime,
		IsPublished:       false,
		CoordinatorIDs:    models.PatientAssignedArray(req.CoordinatorIDs),
		DeclarationForms:  declarationForms,
	}

	if err := tx.Create(&eConsentForm).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create e-consent form", nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
		return
	}

	// Get the created e-consent form with relationships
	var createdForm models.EConsentForm
	if err := config.DB.Preload("Research.CreatedByUser").
		Preload("Video").
		Preload("Declarations.Declaration").
		Where("id = ?", eConsentForm.ID).
		First(&createdForm).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch created e-consent form", nil)
		return
	}

	// Get coordinator info
	coordinators, err := getUsersInfo(createdForm.CoordinatorIDs)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch coordinators", nil)
		return
	}

	// Get patient info
	patients, err := getUsersInfo(createdForm.PatientAssigned)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch patients", nil)
		return
	}

	// Prepare response
	// Encrypt the research creator's username before including in the response
	creatorName := createdForm.Research.CreatedByUser.FirstName + " " + createdForm.Research.CreatedByUser.LastName
	encryptedUserName, err := utils.Encrypt(creatorName)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to encrypt research creator name", nil)
		return
	}

	response := EConsentFormResponse{
		ID:         createdForm.ID,
		ResearchID: createdForm.ResearchID,
		Research: ResearchResponseForConsent{
			ID:          createdForm.Research.ID,
			Title:       createdForm.Research.Title,
			Description: createdForm.Research.Description,
			CategoryID:  createdForm.Research.CategoryID,
			IsPublished: createdForm.Research.IsPublished,
			UserName:    encryptedUserName,
		},
		Title:             createdForm.Title,
		VideoID:           createdForm.VideoID,
		VideoDescription:  createdForm.VideoDescription,
		PersonalInfoCheck: createdForm.PersonalInfoCheck,
		Coordinators:      coordinators,
		PatientAssigned:   patients,
		ExpiryTime:        createdForm.ExpiryTime.UTC().Format(time.RFC3339),
		Video: VideoResponseForConsent{
			ID:          createdForm.Video.ID,
			Title:       createdForm.Video.Title,
			Description: createdForm.Video.Description,
			FilePath:    createdForm.Video.FilePath,
		},
		IsPublished: createdForm.IsPublished,
		CreatedAt:   createdForm.CreatedAt,
		UpdatedAt:   createdForm.UpdatedAt,
	}

	// Add declarations to response if any exist
	if len(createdForm.Declarations) > 0 {
		declarations := make([]ConsentDeclarationResponse, len(createdForm.Declarations))
		for i, decl := range createdForm.Declarations {
			declarations[i] = ConsentDeclarationResponse{
				ID:    decl.Declaration.ID,
				Label: decl.Declaration.Title,
			}
		}
		response.Declarations = declarations
	}

	// Add declaration forms and their declarations to the response (new structure)
	var declarationFormsResponse []map[string]interface{}
	if err := config.DB.Preload("DeclarationForms.Declarations").First(&createdForm, createdForm.ID).Error; err == nil {
		log.Printf("Loaded DeclarationForms: %+v", createdForm.DeclarationForms)
		for _, form := range createdForm.DeclarationForms {
			log.Printf("Form %d Declarations: %+v", form.ID, form.Declarations)
			var declarations []map[string]interface{}
			declarationFormsResponse = append(declarationFormsResponse, map[string]interface{}{
				"id":           form.ID,
				"label":        form.Name,
				"description":  form.Description,
				"declarations": declarations,
			})
		}
	}
	// Convert response struct to map using JSON marshal/unmarshal
	var responseMap map[string]interface{}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to marshal response", nil)
		return
	}
	if err := json.Unmarshal(responseBytes, &responseMap); err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to unmarshal response", nil)
		return
	}
	responseMap["declaration_forms"] = declarationFormsResponse
	utils.JSONResponse(c, http.StatusCreated, "E-consent form created successfully", responseMap)
}

// ListEConsentForms retrieves a list of e-consent forms with filtering, pagination, and sorting
func ListEConsentForms(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can view e-consent forms", nil)
		return
	}

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
	validSortColumns := []string{"title", "is_published", "expiry_time", "created_at", "updated_at"}
	if val := strings.ToLower(query.Get("sort_column")); val != "" {
		if utils.StringInSlice(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortColumnParameter, nil)
			return
		}
	}

	// Parse 'status' filter
	status := "all"
	if val := strings.ToLower(query.Get("status")); val == "draft" || val == "published" || val == "all" {
		status = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid status parameter", nil)
		return
	}

	// Optional 'search_text'
	searchText := strings.TrimSpace(query.Get("search_text"))

		// URL decode the search text to handle encoded spaces and special characters
	if searchText != "" {
		originalSearchText := searchText
		if decoded, err := url.QueryUnescape(searchText); err == nil {
			searchText = strings.TrimSpace(decoded)
			if originalSearchText != searchText {
				fmt.Printf("DEBUG: URL decoded '%s' to '%s'\n", originalSearchText, searchText)
			}
		}
		
		// Convert search text to lowercase for case-insensitive search
		searchText = strings.ToLower(searchText)
	}
	
	// Initialize GORM query
	db := config.DB.Model(&models.EConsentForm{}).
		Where("e_consent_forms.is_deleted = ?", false)
	
	// Debug: Check total econsent forms in database (before any search filter)
	var totalEConsentFormsInDB int64
	config.DB.Model(&models.EConsentForm{}).Where("is_deleted = ?", false).Count(&totalEConsentFormsInDB)
	fmt.Printf("DEBUG: Total econsent forms in database: %d\n", totalEConsentFormsInDB)

	// Apply status filter
	if status == "draft" {
		db = db.Where("e_consent_forms.is_published = ?", false)
	} else if status == "published" {
		db = db.Where("e_consent_forms.is_published = ?", true)
	}

	// Parse search text for date patterns using the utility function
	dateResult := utils.ParseSearchTextForDate(searchText)
	
	// Apply search filter
	if searchText != "" {
		// Build comprehensive search query for title and date
		searchPattern := "%" + searchText + "%"
		
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
				dateConditions = append(dateConditions, "(e_consent_forms.created_at >= ? AND e_consent_forms.created_at < ?)")
				dateArgs = append(dateArgs, startOfDay, endOfDay)
			} else if len(dateResult.Dates) > 0 {
				// Multiple dates - create OR conditions for each date
				for _, date := range dateResult.Dates {
					startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
					endOfDay := startOfDay.Add(24 * time.Hour)
					dateConditions = append(dateConditions, "(e_consent_forms.created_at >= ? AND e_consent_forms.created_at < ?)")
					dateArgs = append(dateArgs, startOfDay, endOfDay)
				}
			}
			
			// Build the combined search query
			combinedSearch := `
			(
				LOWER(e_consent_forms.title) LIKE ?
				)
				`
			
			// Add date conditions if any
			if len(dateConditions) > 0 {
				combinedSearch += " OR " + strings.Join(dateConditions, " OR ")
			}
			
			// Combine text args with date args
			allArgs := append([]interface{}{searchPattern}, dateArgs...)
			db = db.Where(combinedSearch, allArgs...)
		} else if dateResult.IsDayOnly {
			// Day-only search: include day-of-month filter
			combinedSearch := `
				LOWER(e_consent_forms.title) LIKE ? OR
				EXTRACT(DAY FROM e_consent_forms.created_at) = ?
			`
			db = db.Where(combinedSearch, searchPattern, *dateResult.Day)
		} else {
			// Text search only
			combinedSearch := `
				LOWER(e_consent_forms.title) LIKE ?
			`
			db = db.Where(combinedSearch, searchPattern)
		}
	}

	// Get total records count
	var totalRecords int64
	if err := db.Count(&totalRecords).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCountRecords, nil)
		return
	}
	
	// Debug: Log the total count
	fmt.Printf("DEBUG: Total econsent records found: %d\n", totalRecords)

	// Calculate total pages
	var totalPages int
	if totalRecords == 0 {
		totalPages = 0
	} else {
		totalPages = int((totalRecords + int64(perPage) - 1) / int64(perPage))
	}

	// Apply sorting
	db = db.Order("e_consent_forms." + sortColumn + " " + sort)

	// Apply pagination
	offset := (page - 1) * perPage
	db = db.Limit(perPage).Offset(offset)

	// Fetch records with relationships
	var eConsentForms []models.EConsentForm
	if err := db.Preload("Research.CreatedByUser").
		Preload("Video").
		Preload("Declarations.Declaration").
		Preload("DeclarationForms").
		Find(&eConsentForms).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}
	
	// Debug: Log the actual records fetched
	fmt.Printf("DEBUG: Fetched %d econsent records after pagination\n", len(eConsentForms))

	// Collect all user IDs to fetch their info in one go
	allUserIDs := make([]uint, 0)
	for _, form := range eConsentForms {
		allUserIDs = append(allUserIDs, form.CoordinatorIDs...)
		allUserIDs = append(allUserIDs, form.PatientAssigned...)
	}

	// Fetch all user info
	var users []models.User
	if len(allUserIDs) > 0 {
		if err := config.DB.Where("id IN (?)", allUserIDs).Find(&users).Error; err != nil {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch user information", nil)
			return
		}
	}

	// Create a map for quick lookup
	userInfoMap := make(map[uint]UserInfoResponse)
	for _, user := range users {
		userInfoMap[user.ID] = UserInfoResponse{
			ID:    user.ID,
			Label: user.FirstName + " " + user.LastName,
		}
	}

	// Prepare e-consent form responses
	var eConsentFormResponses []EConsentFormResponse
	for _, form := range eConsentForms {
		// Prepare research response
		// Encrypt the research creator's username before including in the response
		creatorName := form.Research.CreatedByUser.FirstName + " " + form.Research.CreatedByUser.LastName
		encryptedUserName, err := utils.Encrypt(creatorName)
		if err != nil {
			encryptedUserName = creatorName // fallback to plain if encryption fails
		}
		researchResponse := ResearchResponseForConsent{
			ID:          form.Research.ID,
			Title:       form.Research.Title,
			Description: form.Research.Description,
			CategoryID:  form.Research.CategoryID,
			IsPublished: form.Research.IsPublished,
			UserName:    encryptedUserName,
		}

		// Prepare video response
		videoResponse := VideoResponseForConsent{
			ID:          form.Video.ID,
			Title:       form.Video.Title,
			Description: form.Video.Description,
			FilePath:    form.Video.FilePath,
		}

		// Prepare coordinators response
		coordinators := make([]UserInfoResponse, 0, len(form.CoordinatorIDs))
		for _, id := range form.CoordinatorIDs {
			if info, ok := userInfoMap[id]; ok {
				encryptedLabel, err := utils.Encrypt(info.Label)
				if err != nil {
					// If encryption fails, fallback to plain label
					encryptedLabel = info.Label
				}
				coordinators = append(coordinators, UserInfoResponse{ID: info.ID, Label: encryptedLabel})
			}
		}

		// Prepare patients response
		patients := make([]UserInfoResponse, 0, len(form.PatientAssigned))
		for _, id := range form.PatientAssigned {
			if info, ok := userInfoMap[id]; ok {
				encryptedLabel, err := utils.Encrypt(info.Label)
				if err != nil {
					// If encryption fails, fallback to plain label
					encryptedLabel = info.Label
				}
				patients = append(patients, UserInfoResponse{ID: info.ID, Label: encryptedLabel})
			}
		}

		// // Prepare declarations response (from Declaration table)
		// declarationsResponse := make([]ConsentDeclarationResponse, 0, len(form.Declarations))
		// for _, decl := range form.Declarations {
		// 	declarationsResponse = append(declarationsResponse, ConsentDeclarationResponse{
		// 		ID:    decl.Declaration.ID,
		// 		Label: decl.Declaration.Title,
		// 	})
		// }

		// Prepare declaration_forms as [{id, label}] for each attached declaration form
		declarationForms := make([]map[string]interface{}, 0, len(form.DeclarationForms))
		for _, df := range form.DeclarationForms {
			declarationForms = append(declarationForms, map[string]interface{}{
				"id":    df.ID,
				"label": df.Name,
			})
		}

		// Check if any patient has submitted their consent for this form
		// Determine if any assigned patient has submitted for the latest version of this e-consent form
		var userHasActed bool
		var submissionCount int64

		// Find the latest version for this e-consent form
		var latestVersionID uint
		config.DB.Model(&models.EConsentFormVersion{}).
			Where("e_consent_form_id = ?", form.ID).
			Select("id").
			Order("version_number DESC").
			Limit(1).
			Scan(&latestVersionID)

		// If there is a version, check submissions for that version
		if latestVersionID > 0 {
			if err := config.DB.Model(&models.PatientEConsent{}).
				Where("e_consent_form_id = ? AND e_consent_form_version_id = ? AND status IN (?) AND is_deleted = ?", form.ID, latestVersionID, []string{"submitted", "approved", "withdrawn", "reviewed", "in_progress", "expired", "archived"}, false).
				Count(&submissionCount).Error; err == nil && submissionCount > 0 {
				userHasActed = true
			} else {
				// Also check if there's an active assignment for this version
				var assignmentCount int64
				if err := config.DB.Model(&models.EConsentAssignment{}).
					Where("e_consent_form_id = ? AND e_consent_form_version_id = ? AND patient_id = ? AND status IN (?) AND is_deleted = ?", 
						form.ID, latestVersionID, user.ID, []string{"submitted", "approved", "withdrawn", "reviewed", "in_progress", "expired"}, false).
					Count(&assignmentCount).Error; err == nil && assignmentCount > 0 {
					userHasActed = true
				} else {
					// Check if there are any PatientEConsent records for this version that indicate the user has acted
					var consentCount int64
					if err := config.DB.Model(&models.PatientEConsent{}).
						Where("e_consent_form_id = ? AND e_consent_form_version_id = ? AND patient_id = ? AND status IN (?) AND is_deleted = ?", 
							form.ID, latestVersionID, user.ID, []string{"submitted", "approved", "withdrawn", "reviewed", "in_progress", "expired", "archived"}, false).
						Count(&consentCount).Error; err == nil && consentCount > 0 {
						userHasActed = true
					} else {
						userHasActed = false
					}
				}
			}
		} else {
			// fallback to previous logic if no version exists
			if err := config.DB.Model(&models.PatientEConsent{}).
				Where("e_consent_form_id = ? AND status IN (?) AND is_deleted = ?", form.ID, []string{"submitted", "approved", "withdrawn", "reviewed", "in_progress", "expired", "archived"}, false).
				Count(&submissionCount).Error; err == nil && submissionCount > 0 {
				userHasActed = true
			} else {
				userHasActed = false
			}
		}

		// Fetch latest version number for this form
		var versionNumber int
		var versionCount int64
		// First check if any versions exist
		if err := config.DB.Model(&models.EConsentFormVersion{}).
			Where("e_consent_form_id = ?", form.ID).
			Count(&versionCount).Error; err != nil {
			// Log the error but don't fail the request
			log.Printf("Failed to check existing versions for form %d: %v", form.ID, err)
			versionNumber = 0
		} else if versionCount > 0 {
			// If versions exist, get the latest version number
			if err := config.DB.Model(&models.EConsentFormVersion{}).
				Where("e_consent_form_id = ?", form.ID).
				Select("MAX(version_number)").
				Scan(&versionNumber).Error; err != nil {
				// Log the error but don't fail the request
				log.Printf("Failed to get latest version number for form %d: %v", form.ID, err)
				versionNumber = 0
			}
		} else {
			// If no versions exist, start with version 0
			versionNumber = 0
		}

		// add a logic for a status to tell whether the form is expired or not 
		isExpired := false
		if !form.ExpiryTime.IsZero() && time.Now().After(form.ExpiryTime) {
			isExpired = true
		}

		eConsentFormResponse := EConsentFormResponse{
			ID:                form.ID,
			ResearchID:        form.ResearchID,
			Research:          researchResponse,
			Title:             form.Title,
			VideoID:           form.VideoID,
			VideoDescription:  form.VideoDescription,
			PersonalInfoCheck: form.PersonalInfoCheck,
			Coordinators:      coordinators,
			PatientAssigned:   patients,
			ExpiryTime:        form.ExpiryTime.UTC().Format(time.RFC3339),
			Video:             videoResponse,
			IsPublished:       form.IsPublished,
			CreatedAt:         form.CreatedAt,
			UpdatedAt:         form.UpdatedAt,
			// Declarations:      declarationsResponse,
			Submitted:         userHasActed,
			DeclarationForms: declarationForms,
			VersionNumber:     versionNumber,
			Expired: 						isExpired,
		}

		eConsentFormResponses = append(eConsentFormResponses, eConsentFormResponse)
	}

	// Prepare the response
	response := EConsentFormListResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      eConsentFormResponses,
	}

	// Send the response
	utils.JSONResponse(c, http.StatusOK, "E-consent forms fetched successfully", response)
}

// DeleteEConsentForm deletes an e-consent form if it's not published
func DeleteEConsentForm(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can delete e-consent forms", nil)
		return
	}

	eConsentFormID := c.Param("id")
	if eConsentFormID == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "E-consent form ID is required", nil)
		return
	}

	// Convert ID to uint
	formID, err := strconv.ParseUint(eConsentFormID, 10, 32)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid e-consent form ID", nil)
		return
	}

	// Check if e-consent form exists and is not deleted
	var eConsentForm models.EConsentForm
	if err := config.DB.Where("id = ? AND is_deleted = ?", formID, false).First(&eConsentForm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusNotFound, "E-consent form not found", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch e-consent form", nil)
		}
		return
	}

	// Check if e-consent form is published
	if eConsentForm.IsPublished {
		utils.JSONResponse(c, http.StatusConflict, "Cannot delete published e-consent form", nil)
		return
	}

	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Delete related consent declarations
	if err := tx.Where("e_consent_form_id = ?", formID).Delete(&models.ConsentDeclaration{}).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to delete consent declarations", nil)
		return
	}

	// Soft delete the e-consent form
	if err := tx.Model(&eConsentForm).Update("is_deleted", true).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to delete e-consent form", nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, "E-consent form deleted successfully", gin.H{
		"id": eConsentForm.ID,
	})
}

// EditEConsentForm edits an e-consent form if it's not published
func EditEConsentForm(c *gin.Context) {
	// Get user from context using the middleware helper
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can edit e-consent forms", nil)
		return
	}

	eConsentFormID := c.Param("id")
	if eConsentFormID == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "E-consent form ID is required", nil)
		return
	}

	// Convert ID to uint
	formID, err := strconv.ParseUint(eConsentFormID, 10, 32)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid e-consent form ID", nil)
		return
	}

	// Check if e-consent form exists and is not deleted
	var eConsentForm models.EConsentForm
	if err := config.DB.Where("id = ? AND is_deleted = ?", formID, false).First(&eConsentForm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusNotFound, "E-consent form not found", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch e-consent form", nil)
		}
		return
	}

	// Check if e-consent form is published
	// if eConsentForm.IsPublished {
	// 	utils.JSONResponse(c, http.StatusConflict, "Cannot edit published e-consent form", nil)
	// 	return
	// }

	// ------- Proceed with new versions --------- 

	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start database transaction", nil)
		return
	}

	// Find all assignments for this form
	var assignments []models.EConsentAssignment
	tx.Where("e_consent_form_id = ? AND is_deleted = ?", formID, false).Find(&assignments)

	log.Printf("Assignments: %+v", assignments)

	// Separate patients who have submitted and not submitted
	submittedPatientIDs := make(map[uint]bool)
	notSubmittedPatientIDs := make(map[uint]bool)
	for _, a := range assignments {
		log.Printf("Assignment: ID=%d, PatientID=%d, Status=%s", a.ID, a.PatientID, a.Status)
		switch a.Status {
		case models.EConsentStatusSubmitted, models.EConsentStatusApproved, models.EConsentStatusWithdrawn, models.EConsentStatusReviewed:
			submittedPatientIDs[a.PatientID] = true
		default:
			notSubmittedPatientIDs[a.PatientID] = true
		}
	}
	log.Printf("SubmittedPatientIDs: %+v", submittedPatientIDs)
	log.Printf("NotSubmittedPatientIDs: %+v", notSubmittedPatientIDs)

	// Bind request data
	var req EditEConsentFormRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate expiry time is in the future
	if req.ExpiryTime.Before(time.Now()) {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Expiry time must be in the future", nil)
		return
	}

	// If any patient has submitted (and not withdrawn), check if this is only a patient addition
	// We exclude withdrawn patients from the "submitted" count for version creation logic
	activeSubmittedCount := len(submittedPatientIDs)
	
	// Check if form has been published (has versions)
	var versionCount int64
	tx.Model(&models.EConsentFormVersion{}).Where("e_consent_form_id = ?", eConsentForm.ID).Count(&versionCount)
	
	// If no patients have submitted AND no versions exist, do simple update
	if activeSubmittedCount == 0 && versionCount == 0 {
		log.Printf("No patients have submitted - performing simple update")
		
		// Update the main eConsentForm fields
		updates := map[string]interface{}{
			"research_id":         req.ResearchID,
			"title":               req.Title,
			"video_id":            req.VideoID,
			"video_description":   req.VideoDescription,
			"personal_info_check": req.PersonalInfoCheck,
			"expiry_time":         req.ExpiryTime,
		}
		if err := tx.Model(&eConsentForm).Updates(updates).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update e-consent form", nil)
			return
		}
		if err := tx.Model(&eConsentForm).Update("patient_assigned", models.PatientAssignedArray(req.PatientAssigned)).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update patient assignments", nil)
			return
		}
		if err := tx.Model(&eConsentForm).Update("coordinator_ids", models.PatientAssignedArray(req.CoordinatorIDs)).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update coordinator assignments", nil)
			return
		}
		if req.DeclarationFormIDs != nil {
			// Validate all declaration forms exist
			var declarationForms []models.DeclarationForm
			if err := tx.Preload("Declarations").Where("id IN (?)", req.DeclarationFormIDs).Find(&declarationForms).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
				return
			}
			if len(declarationForms) != len(req.DeclarationFormIDs) {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
				return
			}
			err := tx.Model(&eConsentForm).Association("DeclarationForms").Replace(declarationForms)
			if err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update declaration forms", nil)
				return
			}
		}

		// Handle patient assignments for simple update
		// Remove assignments for patients no longer in the list
		oldAssignments := make(map[uint]models.EConsentAssignment)
		for _, a := range assignments {
			oldAssignments[a.PatientID] = a
		}

		// Mark assignments as deleted for removed patients
		for _, assignment := range assignments {
			found := false
			for _, patientID := range req.PatientAssigned {
				if assignment.PatientID == patientID {
					found = true
					break
				}
			}
			if !found {
				// Patient was removed, mark assignment as deleted
				if err := tx.Model(&assignment).Update("is_deleted", true).Error; err != nil {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusInternalServerError, "Failed to remove patient assignment", nil)
					return
				}
			}
		}

		// Create new assignments for new patients
		for _, patientID := range req.PatientAssigned {
			if _, exists := oldAssignments[patientID]; !exists {
				// Check if versions exist for this form
				var versionCount int64
				if err := tx.Model(&models.EConsentFormVersion{}).Where("e_consent_form_id = ?", eConsentForm.ID).Count(&versionCount).Error; err != nil {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusInternalServerError, "Failed to check version count", nil)
					return
				}

				if versionCount > 0 {
					// Get the latest version for new assignments
					var latestVersion models.EConsentFormVersion
					err = tx.Where("e_consent_form_id = ?", eConsentForm.ID).Order("version_number DESC").First(&latestVersion).Error
					if err != nil {
						tx.Rollback()
						utils.JSONResponse(c, http.StatusInternalServerError, "Failed to get latest version for new patient assignment", nil)
						return
					}

					// Create new assignment for new patient with version
					newAssignment := models.EConsentAssignment{
						EConsentFormID:        eConsentForm.ID,
						PatientID:             patientID,
						AssignedAt:            time.Now(),
						Status:                models.EConsentStatusAssigned,
						IsDeleted:             false,
						EConsentFormVersionID: latestVersion.ID,
					}
					if err := tx.Create(&newAssignment).Error; err != nil {
						tx.Rollback()
						utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create new patient assignment", nil)
						return
					}
				} else {
					// No versions exist yet, skip creating assignment for now
					// Assignment will be created when the form is first published and versions are created
					log.Printf("Skipping assignment creation for patient %d - no versions exist yet for form %d", patientID, eConsentForm.ID)
				}
			}
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
			return
		}

		utils.JSONResponse(c, http.StatusOK, "E-consent form updated successfully", nil)
		return
	}
	
	// If no patients have submitted BUT form has been published (has versions), update the latest version
	if activeSubmittedCount == 0 && versionCount > 0 {
		log.Printf("Form has been published but no patients have submitted - updating latest version")
		
		// Get the latest version
		var latestVersion models.EConsentFormVersion
		if err := tx.Where("e_consent_form_id = ?", eConsentForm.ID).Order("version_number DESC").First(&latestVersion).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to get latest version", nil)
			return
		}
		
		// Update the main eConsentForm fields
		updates := map[string]interface{}{
			"research_id":         req.ResearchID,
			"title":               req.Title,
			"video_id":            req.VideoID,
			"video_description":   req.VideoDescription,
			"personal_info_check": req.PersonalInfoCheck,
			"expiry_time":         req.ExpiryTime,
		}
		if err := tx.Model(&eConsentForm).Updates(updates).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update e-consent form", nil)
			return
		}
		if err := tx.Model(&eConsentForm).Update("patient_assigned", models.PatientAssignedArray(req.PatientAssigned)).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update patient assignments", nil)
			return
		}
		if err := tx.Model(&eConsentForm).Update("coordinator_ids", models.PatientAssignedArray(req.CoordinatorIDs)).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update coordinator assignments", nil)
			return
		}
		if req.DeclarationFormIDs != nil {
					// Validate all declaration forms exist
		var declarationForms []models.DeclarationForm
		if err := tx.Preload("Declarations").Where("id IN (?)", req.DeclarationFormIDs).Find(&declarationForms).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
			return
		}
			if len(declarationForms) != len(req.DeclarationFormIDs) {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
				return
			}
			err := tx.Model(&eConsentForm).Association("DeclarationForms").Replace(declarationForms)
			if err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update declaration forms", nil)
				return
			}
		}
		
		// For published forms with no submissions, we need to update both the main form fields
		// and the version snapshot to ensure patient portal shows updated content
		
		// Update the latest version's snapshot with new content
		// Fetch required data for new snapshot
		var research models.Research
		if err := tx.Preload("Documents").Where("id = ?", req.ResearchID).First(&research).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to load research", nil)
			return
		}
		
		var video models.Video
		if err := tx.Where("id = ?", req.VideoID).First(&video).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to load video", nil)
			return
		}
		
		// Build declaration forms snapshot
		var formsSnapshot []map[string]interface{}
		if req.DeclarationFormIDs != nil {
			var declarationForms []models.DeclarationForm
			if err := tx.Preload("Declarations").Where("id IN (?)", req.DeclarationFormIDs).Find(&declarationForms).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to load declaration forms", nil)
				return
			}
			
			for _, form := range declarationForms {
				var statements []map[string]interface{}
				for _, decl := range form.Declarations {
					statements = append(statements, map[string]interface{}{
						"id":            decl.ID,
						"title":         decl.Title,
						"statement":     decl.Statement,
						"options":       decl.Options,
						"response_type": decl.ResponseType,
					})
				}
				formsSnapshot = append(formsSnapshot, map[string]interface{}{
					"id":          form.ID,
					"name":        form.Name,
					"description": form.Description,
					"statements":  statements,
				})
			}
		}
		
		// Build research document snapshot
		var researchDocSnap map[string]interface{}
		if len(research.Documents) > 0 {
			doc := research.Documents[0]
			researchDocSnap = map[string]interface{}{
				"id":        doc.ID,
				"file_name": filepath.Base(doc.FilePath),
				"file_type": doc.FileType,
			}
		}
		
		// Create new snapshot
		snapshot := map[string]interface{}{
			"research": map[string]interface{}{
				"id":          research.ID,
				"title":       research.Title,
				"description": research.Description,
				"category_id": research.CategoryID,
				"is_published": research.IsPublished,
				"mimeType":    "application/pdf",
				"document":    researchDocSnap,
			},
			"video": map[string]interface{}{
				"id":          video.ID,
				"title":       video.Title,
				"description": video.Description,
				"file_path":   video.FilePath,
				"mimeType":    video.MimeType,
			},
			"video_description": req.VideoDescription,
			"declaration_forms": formsSnapshot,
		}
		snapshotJSON, err := json.Marshal(snapshot)
		if err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to serialize e-consent version snapshot", nil)
			return
		}
		
		// Update the latest version with new snapshot
		latestVersion.SnapshotJSON = string(snapshotJSON)
		if err := tx.Save(&latestVersion).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update version snapshot", nil)
			return
		}
		
		// Handle patient assignments - DO NOT update existing assignments to point to the updated version
		// This preserves the audit trail by keeping assignments pointing to their original versions
		oldAssignments := make(map[uint]models.EConsentAssignment)
		for _, a := range assignments {
			oldAssignments[a.PatientID] = a
		}
		
		// IMPORTANT: Do NOT update existing assignments' version IDs
		// This prevents the bug where V1 assignments get changed to V2
		// Each assignment should maintain its historical version for proper audit trails
		
		// Log the assignments to help with debugging
		log.Printf("DEBUG: Preserving existing assignments with their original versions:")
		for _, assignment := range assignments {
			log.Printf("  Assignment ID: %d, Patient: %d, Version: %d, Status: %s", 
				assignment.ID, assignment.PatientID, assignment.EConsentFormVersionID, assignment.Status)
		}
		
		// Handle removed patients
		for _, assignment := range assignments {
			found := false
			for _, patientID := range req.PatientAssigned {
				if assignment.PatientID == patientID {
					found = true
					break
				}
			}
			if !found {
				// Patient was removed, mark assignment as deleted
				if err := tx.Model(&assignment).Update("is_deleted", true).Error; err != nil {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusInternalServerError, "Failed to remove patient assignment", nil)
					return
				}
			}
		}
		
		// Create new assignments for new patients
		for _, patientID := range req.PatientAssigned {
			if _, exists := oldAssignments[patientID]; !exists {
				// Create new assignment for new patient
				newAssignment := models.EConsentAssignment{
					EConsentFormID:        eConsentForm.ID,
					PatientID:             patientID,
					AssignedAt:            time.Now(),
					Status:                models.EConsentStatusAssigned,
					IsDeleted:             false,
					EConsentFormVersionID: latestVersion.ID,
				}
				if err := tx.Create(&newAssignment).Error; err != nil {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create new patient assignment", nil)
					return
				}
			}
		}
		
		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
			return
		}
		
		utils.JSONResponse(c, http.StatusOK, "E-consent form updated successfully", nil)
		return
	}
	
	// If patients have submitted, check if this is only a patient addition
	if activeSubmittedCount > 0 {
		// Check if this is only an expiry date update (no other changes)
		isOnlyExpiryUpdate := req.ResearchID == eConsentForm.ResearchID &&
			req.Title == eConsentForm.Title &&
			req.VideoID == eConsentForm.VideoID &&
			req.VideoDescription == eConsentForm.VideoDescription &&
			req.PersonalInfoCheck == eConsentForm.PersonalInfoCheck &&
			req.ExpiryTime != eConsentForm.ExpiryTime // Only expiry time changed
		
		// Check if patient assignments are the same
		if len(req.PatientAssigned) != len(eConsentForm.PatientAssigned) {
			isOnlyExpiryUpdate = false
		} else {
			// Compare patient assignments
			oldPatientSet := make(map[uint]bool)
			for _, pid := range eConsentForm.PatientAssigned {
				oldPatientSet[pid] = true
			}
			for _, pid := range req.PatientAssigned {
				if !oldPatientSet[pid] {
					isOnlyExpiryUpdate = false
					break
				}
			}
		}
		
		// Check if coordinator assignments are the same
		if len(req.CoordinatorIDs) != len(eConsentForm.CoordinatorIDs) {
			isOnlyExpiryUpdate = false
		} else {
			oldCoordinatorSet := make(map[uint]bool)
			for _, cid := range eConsentForm.CoordinatorIDs {
				oldCoordinatorSet[cid] = true
			}
			for _, cid := range req.CoordinatorIDs {
				if !oldCoordinatorSet[cid] {
					isOnlyExpiryUpdate = false
					break
				}
			}
		}
		
		// Check if declaration forms are the same
		if req.DeclarationFormIDs != nil {
			// Get current declaration form IDs using a direct query
			var currentDeclarationFormIDs []uint
			if err := tx.Raw("SELECT declaration_form_id FROM econsent_form_declaration_forms WHERE e_consent_form_id = ?", eConsentForm.ID).Scan(&currentDeclarationFormIDs).Error; err != nil {
				log.Printf("Failed to get current declaration form IDs: %v", err)
				isOnlyExpiryUpdate = false
			} else {
				// Compare declaration form IDs
				if len(req.DeclarationFormIDs) != len(currentDeclarationFormIDs) {
					isOnlyExpiryUpdate = false
				} else {
					currentSet := make(map[uint]bool)
					for _, id := range currentDeclarationFormIDs {
						currentSet[id] = true
					}
					for _, id := range req.DeclarationFormIDs {
						if !currentSet[id] {
							isOnlyExpiryUpdate = false
							break
						}
					}
				}
			}
		} else {
			// If no declaration forms in request, check if current form has any
			var count int64
			if err := tx.Raw("SELECT COUNT(*) FROM econsent_form_declaration_forms WHERE e_consent_form_id = ?", eConsentForm.ID).Scan(&count).Error; err == nil && count > 0 {
				isOnlyExpiryUpdate = false
			}
		}
		
		// If only expiry date is being updated, do a simple update instead of versioning
		if isOnlyExpiryUpdate {
			log.Printf("Only expiry date update detected - performing simple update without versioning")
			
			// Update only the expiry time
			if err := tx.Model(&eConsentForm).Update("expiry_time", req.ExpiryTime).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update expiry time", nil)
				return
			}
			
			// Update the latest version's snapshot to reflect the new expiry time
			var latestVersion models.EConsentFormVersion
			if err := tx.Where("e_consent_form_id = ?", eConsentForm.ID).Order("version_number DESC").First(&latestVersion).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to get latest version for expiry update", nil)
				return
			}
			
			// Update the snapshot to include the new expiry time
			// We need to update the snapshot JSON to reflect the new expiry time
			// This ensures the patient portal shows the updated expiry time
			if err := tx.Model(&latestVersion).Update("updated_at", time.Now()).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update version timestamp", nil)
				return
			}
			
			if err := tx.Commit().Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
				return
			}
			
			utils.JSONResponse(c, http.StatusOK, "E-consent form expiry time updated successfully (no new version created)", nil)
			return
		}
		
		// If is_only_patient_addition is true, don't create a new version
		if req.IsOnlyPatientAddition {
			// Just update the existing form with new patients and skip version creation
			log.Printf("Patient addition only - skipping version creation")
			
			// Update the main eConsentForm fields
			updates := map[string]interface{}{
				"research_id":         req.ResearchID,
				"title":               req.Title,
				"video_id":            req.VideoID,
				"video_description":   req.VideoDescription,
				"personal_info_check": req.PersonalInfoCheck,
				"expiry_time":         req.ExpiryTime,
			}
			if err := tx.Model(&eConsentForm).Updates(updates).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update e-consent form", nil)
				return
			}
			if err := tx.Model(&eConsentForm).Update("patient_assigned", models.PatientAssignedArray(req.PatientAssigned)).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update patient assignments", nil)
				return
			}
			if err := tx.Model(&eConsentForm).Update("coordinator_ids", models.PatientAssignedArray(req.CoordinatorIDs)).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update coordinator assignments", nil)
				return
			}
			if req.DeclarationFormIDs != nil {
				// Validate all declaration forms exist
				var declarationForms []models.DeclarationForm
				if err := tx.Preload("Declarations").Where("id IN (?)", req.DeclarationFormIDs).Find(&declarationForms).Error; err != nil {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
					return
				}
				if len(declarationForms) != len(req.DeclarationFormIDs) {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
					return
				}
				err := tx.Model(&eConsentForm).Association("DeclarationForms").Replace(declarationForms)
				if err != nil {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update declaration forms", nil)
					return
				}
			}

			// Fetch research for notifications
			var research models.Research
			if err := tx.Preload("Documents").Where("id = ?", req.ResearchID).First(&research).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to load research", nil)
				return
			}

			// Get the latest version for new assignments
			var latestVersion models.EConsentFormVersion
			err = tx.Where("e_consent_form_id = ?", eConsentForm.ID).Order("version_number DESC").First(&latestVersion).Error
			if err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to get latest version for patient assignments", nil)
				return
			}

			// Create new assignments for new patients only (use the latest existing version)
			var patientsToNotify []models.User
			oldAssignments := make(map[uint]models.EConsentAssignment)
			for _, a := range assignments {
				oldAssignments[a.PatientID] = a
			}

			for _, patientID := range req.PatientAssigned {
				if old, exists := oldAssignments[patientID]; exists {
					// Check if patient had withdrawn from this form
					if old.Status == models.EConsentStatusWithdrawn {
						// Don't reassign withdrawn patients
						continue
					}
					// Patient already has an assignment and is not withdrawn, skip
					continue
				}
				// Create new assignment for new patient using the latest version
				newAssignment := models.EConsentAssignment{
					EConsentFormID:        eConsentForm.ID,
					PatientID:             patientID,
					AssignedAt:            time.Now(),
					Status:                models.EConsentStatusAssigned,
					IsDeleted:             false,
					EConsentFormVersionID: latestVersion.ID,
				}
				tx.Create(&newAssignment)

				// Collect patient for notification after commit
				var patient models.User
				if err := tx.Where("id = ?", patientID).First(&patient).Error; err == nil {
					patientsToNotify = append(patientsToNotify, patient)
				}
			}

			if err := tx.Commit().Error; err != nil {
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
				return
			}

			// Notify only new patients about the assignment
			for _, patient := range patientsToNotify {
				title := "New E-Consent Assigned"
				message := "You have been assigned a new e-consent form: '" + req.Title + "' for research: '" + research.Title + "'. Please review and submit your consent."
				notificationType := "E-Consent-Assign"
				entityType := "econsents"
				entityID := eConsentForm.ID
				username := patient.FirstName + " " + patient.LastName
				metadata := map[string]interface{}{
					"econsent_form_id": eConsentForm.ID,
					"research_id":      req.ResearchID,
					"status":           "assigned",
				}
				err := services.CreateNotification(config.DB, patient.ID, title, message, notificationType, username, entityType, entityID, metadata)
				if err != nil {
					log.Printf("Failed to create notification for patient %d: %v", patient.ID, err)
				}

				// Send email notification
				emailBody := emails.EConsentAssignedEmail(
					patient.FirstName+" "+patient.LastName,
					req.Title,
					research.Title,
				)
				subject := "New E-Consent Assigned"
				err = config.SendEmail([]string{patient.Email}, subject, emailBody)
				if err != nil {
					log.Printf("Failed to send email to patient %d: %v", patient.ID, err)
				}
			}

			utils.JSONResponse(c, http.StatusOK, "E-consent form updated with new patients (no new version created)", nil)
			return
		}

		// --- Create new version logic ---
		// Validate all declaration forms exist
		var declarationForms []models.DeclarationForm
		if req.DeclarationFormIDs != nil {
			if err := tx.Preload("Declarations").Where("id IN (?)", req.DeclarationFormIDs).Find(&declarationForms).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "One or more declaration forms not found", nil)
				return
			}
			if len(declarationForms) != len(req.DeclarationFormIDs) {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
				return
			}
		}

		// Fetch research and video for snapshot
		var research models.Research
		if err := tx.Preload("Documents").Where("id = ?", req.ResearchID).First(&research).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to load research", nil)
			return
		}
		var video models.Video
		if err := tx.Where("id = ?", req.VideoID).First(&video).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to load video", nil)
			return
		}

		// Prepare research document snapshot (first document if exists)
		type ResearchDocumentSnapshot struct {
			ID       uint   `json:"id"`
			FileName string `json:"file_name"`
			FileType string `json:"file_type"`
		}
		var researchDocSnap *ResearchDocumentSnapshot
		if len(research.Documents) > 0 {
			doc := research.Documents[0]
			researchDocSnap = &ResearchDocumentSnapshot{
				ID:       doc.ID,
				FileName: filepath.Base(doc.FilePath),
				FileType: doc.FileType,
			}
		}

		// Prepare declaration forms snapshot
		type DeclarationSnapshot struct {
			ID           uint     `json:"id"`
			Title        string   `json:"title"`
			Statement    string   `json:"statement"`
			Options      []string `json:"options"`
			ResponseType string   `json:"response_type"`
		}
		type DeclarationFormSnapshot struct {
			ID          uint                 `json:"id"`
			Name        string               `json:"name"`
			Description string               `json:"description"`
			Statements  []DeclarationSnapshot `json:"statements"`
		}
		var formsSnapshot []DeclarationFormSnapshot
		for _, df := range declarationForms {
			var statements []DeclarationSnapshot
			for _, decl := range df.Declarations {
				statements = append(statements, DeclarationSnapshot{
					ID:    decl.ID,
					Title: decl.Title,
					Statement:  decl.Statement,
					Options: decl.Options,
					ResponseType: string(decl.ResponseType),
				})
			}
			formsSnapshot = append(formsSnapshot, DeclarationFormSnapshot{
				ID:          df.ID,
				Name:        df.Name,
				Description: df.Description,
				Statements:  statements,
			})
		}

		type ResearchSnapshot struct {
			ID          uint                   `json:"id"`
			Title       string                 `json:"title"`
			Description string                 `json:"description"`
			CategoryID  uint                   `json:"category_id"`
			IsPublished bool                   `json:"is_published"`
			MimeType    string                 `json:"mimeType"`
			Document    *ResearchDocumentSnapshot `json:"document,omitempty"`
		}

		type VideoSnapshot struct {
			ID          uint   `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			FilePath    string `json:"file_path"`
			MimeType    string `json:"mimeType"`
		}

		type EConsentFormVersionSnapshot struct {
			Research         ResearchSnapshot           `json:"research"`
			Video            VideoSnapshot              `json:"video"`
			VideoDescription string                     `json:"video_description"`
			DeclarationForms []DeclarationFormSnapshot  `json:"declaration_forms"`
		}

		snapshot := EConsentFormVersionSnapshot{
			Research: ResearchSnapshot{
				ID:          research.ID,
				Title:       research.Title,
				Description: research.Description,
				CategoryID:  research.CategoryID,
				IsPublished: research.IsPublished,
				MimeType:    "application/pdf",
				Document:    researchDocSnap,
			},
			Video: VideoSnapshot{
				ID:          video.ID,
				Title:       video.Title,
				Description: video.Description,
				FilePath:    video.FilePath,
				MimeType:    video.MimeType,
			},
			VideoDescription: req.VideoDescription,
			DeclarationForms: formsSnapshot,
		}
		snapshotJSON, err := json.Marshal(snapshot)
		if err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to serialize e-consent version snapshot", nil)
			return
		}

		// Find latest version number
		var latestVersion int
		var versionCount int64
		// First check if any versions exist
		if err := tx.Model(&models.EConsentFormVersion{}).
			Where("e_consent_form_id = ?", formID).
			Count(&versionCount).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to check existing versions", nil)
			return
		}
		
		if versionCount > 0 {
			// If versions exist, get the latest version number
			if err := tx.Model(&models.EConsentFormVersion{}).
				Where("e_consent_form_id = ?", formID).
				Select("MAX(version_number)").
				Scan(&latestVersion).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to get latest version number", nil)
				return
			}
		} else {
			// If no versions exist, start with version 1
			latestVersion = 0
		}
		
		// Always create a new version number to ensure proper versioning
		// This prevents the issue where editing V2 multiple times doesn't create V3
		newVersionNumber := latestVersion + 1

		// Create new version
		version := models.EConsentFormVersion{
			EConsentFormID: eConsentForm.ID,
			VersionNumber:  newVersionNumber,
			SnapshotJSON:   string(snapshotJSON),
			PublishedAt:    time.Now(),
		}
		
		log.Printf("Creating new version: FormID=%d, PreviousVersion=%d, NewVersionNumber=%d", eConsentForm.ID, latestVersion, version.VersionNumber)
		
		if err := tx.Create(&version).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to create e-consent version: %v", err)
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save e-consent version", nil)
			return
		}
		
		log.Printf("Successfully created version with ID: %d", version.ID)

		// After creating the new version, create a new assignment for all patients and archive old assignments if needed
		var patientsToNotify []models.User
		// Build a map of previous assignments by patient ID
		oldAssignments := make(map[uint]models.EConsentAssignment)
		for _, a := range assignments {
			oldAssignments[a.PatientID] = a
		}

		for _, patientID := range req.PatientAssigned {
			if old, exists := oldAssignments[patientID]; exists {
				// Archive old assignment if needed - archive ALL old assignments for this patient
				// This ensures clean separation between versions
				tx.Model(&models.EConsentAssignment{}).
					Where("id = ?", old.ID).
					Update("status", models.EConsentStatusArchived)
			}
			// Always create a new assignment for the new version
			newAssignment := models.EConsentAssignment{
				EConsentFormID:        eConsentForm.ID,
				PatientID:             patientID,
				AssignedAt:            time.Now(),
				Status:                models.EConsentStatusAssigned,
				IsDeleted:             false,
				EConsentFormVersionID: version.ID,
			}
			if err := tx.Create(&newAssignment).Error; err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create new assignment for patient", nil)
				return
			}

			// Collect patient for notification after commit
			var patient models.User
			if err := tx.Where("id = ?", patientID).First(&patient).Error; err == nil {
				patientsToNotify = append(patientsToNotify, patient)
			}
		}
		log.Printf("Patients to notify after version update: %+v", patientsToNotify)

		// Optionally, update the main eConsentForm fields (title, video, etc.)
		updates := map[string]interface{}{
			"research_id":         req.ResearchID,
			"title":               req.Title,
			"video_id":            req.VideoID,
			"video_description":   req.VideoDescription,
			"personal_info_check": req.PersonalInfoCheck,
			"expiry_time":         req.ExpiryTime,
		}
		if err := tx.Model(&eConsentForm).Updates(updates).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update e-consent form", nil)
			return
		}
		if err := tx.Model(&eConsentForm).Update("patient_assigned", models.PatientAssignedArray(req.PatientAssigned)).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update patient assignments", nil)
			return
		}
		if err := tx.Model(&eConsentForm).Update("coordinator_ids", models.PatientAssignedArray(req.CoordinatorIDs)).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update coordinator assignments", nil)
			return
		}
		if req.DeclarationFormIDs != nil {
			err := tx.Model(&eConsentForm).Association("DeclarationForms").Replace(declarationForms)
			if err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update declaration forms", nil)
				return
			}
		}
		if err := tx.Commit().Error; err != nil {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
			return
		}

		// Separate patients to notify: existing (already assigned before) and new (just assigned now)
		// Build a set of previously assigned patient IDs (from assignments)
		prevAssigned := make(map[uint]bool)
		for _, a := range assignments {
			prevAssigned[a.PatientID] = true
		}
		// Build a set of currently assigned patient IDs (from req.PatientAssigned)
		currentAssigned := make(map[uint]bool)
		for _, pid := range req.PatientAssigned {
			currentAssigned[pid] = true
		}

		// Find new patients (in currentAssigned but not in prevAssigned)
		newPatientIDs := make([]uint, 0)
		for pid := range currentAssigned {
			if !prevAssigned[pid] {
				newPatientIDs = append(newPatientIDs, pid)
			}
		}

		// Find existing patients (in both prevAssigned and currentAssigned)
		existingPatientIDs := make([]uint, 0)
		for pid := range currentAssigned {
			if prevAssigned[pid] {
				existingPatientIDs = append(existingPatientIDs, pid)
			}
		}

		// Fetch user info for new and existing patients
		var newPatients, existingPatients []models.User
		if len(newPatientIDs) > 0 {
			if err := config.DB.Where("id IN (?)", newPatientIDs).Find(&newPatients).Error; err != nil {
				log.Printf("Failed to fetch new patients: %v", err)
			}
		}
		if len(existingPatientIDs) > 0 {
			if err := config.DB.Where("id IN (?)", existingPatientIDs).Find(&existingPatients).Error; err != nil {
				log.Printf("Failed to fetch existing patients: %v", err)
			}
		}

		// Notify existing patients about the update
		for _, patient := range existingPatients {
			title := "E-Consent Updated"
			message := "The e-consent form '" + req.Title + "' has been updated. Please review and submit your consent."
			notificationType := "E-Consent-Update"
			entityType := "econsents"
			entityID := eConsentForm.ID
			username := patient.FirstName + " " + patient.LastName
			metadata := map[string]interface{}{
				"econsent_form_id": eConsentForm.ID,
				"research_id":      req.ResearchID,
				"status":           "updated",
			}
			err := services.CreateNotification(config.DB, patient.ID, title, message, notificationType, username, entityType, entityID, metadata)
			if err != nil {
				log.Printf("Failed to create notification for patient %d: %v", patient.ID, err)
			}

			// Send email notification (use EConsentUpdatedEmail)
			emailBody := emails.EConsentUpdatedEmail(
				patient.FirstName+" "+patient.LastName,
				req.Title,
				research.Title,
			)
			subject := "E-Consent Updated"
			err = config.SendEmail([]string{patient.Email}, subject, emailBody)
			if err != nil {
				log.Printf("Failed to send email to patient %d: %v", patient.ID, err)
			}
		}

		// Notify new patients about the assignment
		for _, patient := range newPatients {
			title := "New E-Consent Assigned"
			message := "You have been assigned a new e-consent form: '" + req.Title + "' for research: '" + research.Title + "'. Please review and submit your consent."
			notificationType := "E-Consent-Assign"
			entityType := "econsents"
			entityID := eConsentForm.ID
			username := patient.FirstName + " " + patient.LastName
			metadata := map[string]interface{}{
				"econsent_form_id": eConsentForm.ID,
				"research_id":      req.ResearchID,
				"status":           "assigned",
			}
			err := services.CreateNotification(config.DB, patient.ID, title, message, notificationType, username, entityType, entityID, metadata)
			if err != nil {
				log.Printf("Failed to create notification for patient %d: %v", patient.ID, err)
			}

			// Send email notification (use EConsentAssignedEmail)
			emailBody := emails.EConsentAssignedEmail(
				patient.FirstName+" "+patient.LastName,
				req.Title,
				research.Title,
			)
			subject := "New E-Consent Assigned"
			err = config.SendEmail([]string{patient.Email}, subject, emailBody)
			if err != nil {
				log.Printf("Failed to send email to patient %d: %v", patient.ID, err)
			}
		}
		utils.JSONResponse(c, http.StatusOK, "E-consent form updated and versioned successfully", nil)
		return
	}

	// If no patient has submitted, allow in-place edit as before
	// ... (existing in-place edit logic) ...

	// If no patient has submitted, allow in-place edit as before
	// Verify that the research exists and is not deleted
	var research models.Research
	if err := tx.Where("id = ? AND is_deleted = ?", req.ResearchID, false).First(&research).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid research ID", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to validate research", nil)
		}
		return
	}

	// Verify that the video exists and is not deleted
	var video models.Video
	if err := tx.Where("id = ? AND is_deleted = ?", req.VideoID, false).First(&video).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid video ID", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to validate video", nil)
		}
		return
	}

	// Verify that all coordinators exist and are not deleted
	for _, coordinatorID := range req.CoordinatorIDs {
		var coordinator models.User
		if err := tx.Where("id = ? AND is_deleted = ?", coordinatorID, false).First(&coordinator).Error; err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid coordinator ID: "+strconv.FormatUint(uint64(coordinatorID), 10), nil)
			} else {
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to validate coordinator", nil)
			}
			return
		}
	}

	// Verify that all assigned patients exist and are not deleted
	for _, patientID := range req.PatientAssigned {
		var patient models.User
		if err := tx.Where("id = ? AND is_deleted = ?", patientID, false).First(&patient).Error; err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid patient ID: "+strconv.FormatUint(uint64(patientID), 10), nil)
			} else {
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to validate patient", nil)
			}
			return
		}
	}

	// Check if e-consent form with same title already exists for this research (excluding current form)
	var existingForm models.EConsentForm
	if err := tx.Where("research_id = ? AND LOWER(title) = LOWER(?) AND id != ? AND is_deleted = ?", req.ResearchID, req.Title, formID, false).First(&existingForm).Error; err == nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusConflict, "An e-consent form with this title already exists for this research", nil)
		return
	} else if err != gorm.ErrRecordNotFound {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to check e-consent form title", nil)
		return
	}

	// Update e-consent form
	updates := map[string]interface{}{
		"research_id":         req.ResearchID,
		"title":               req.Title,
		"video_id":            req.VideoID,
		"video_description":   req.VideoDescription,
		"personal_info_check": req.PersonalInfoCheck,
		"expiry_time":         req.ExpiryTime,
	}

	if err := tx.Model(&eConsentForm).Updates(updates).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update e-consent form", nil)
		return
	}

	// Update patient_assigned separately to handle JSON serialization
	if err := tx.Model(&eConsentForm).Update("patient_assigned", models.PatientAssignedArray(req.PatientAssigned)).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update patient assignments", nil)
		return
	}

	// Update coordinator_ids separately to handle JSON serialization
	if err := tx.Model(&eConsentForm).Update("coordinator_ids", models.PatientAssignedArray(req.CoordinatorIDs)).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update coordinator assignments", nil)
		return
	}

	// After updating coordinator_ids and patient_assigned, handle declaration_form_ids
	if req.DeclarationFormIDs != nil {
		// Validate all declaration forms exist
		var declarationForms []models.DeclarationForm
		if err := tx.Where("id IN (?)", req.DeclarationFormIDs).Find(&declarationForms).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
			return
		}
		if len(declarationForms) != len(req.DeclarationFormIDs) {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "One or more declaration forms not found", nil)
			return
		}
		log.Printf("Updating declaration forms for eConsentForm %d: %+v", eConsentForm.ID, req.DeclarationFormIDs)
		err := tx.Model(&eConsentForm).Association("DeclarationForms").Replace(declarationForms)
		log.Printf("Replace error: %v", err)
		if err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update declaration forms", nil)
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
		return
	}

	// Reload the updated eConsentForm with DeclarationForms
	var updatedForm models.EConsentForm
	if err := config.DB.Preload("Research.CreatedByUser").
		Preload("Video").
		Preload("DeclarationForms").
		Where("id = ?", formID).
		First(&updatedForm).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch updated e-consent form", nil)
		return
	}

	// Get coordinator info
	coordinators, err := getUsersInfo(updatedForm.CoordinatorIDs)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch coordinators", nil)
		return
	}

	// Get patient info
	patients, err := getUsersInfo(updatedForm.PatientAssigned)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch patients", nil)
		return
	}

	// Prepare response
	// Encrypt the research creator's username before including in the response
	creatorName := updatedForm.Research.CreatedByUser.FirstName + " " + updatedForm.Research.CreatedByUser.LastName
	encryptedUserName, err := utils.Encrypt(creatorName)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to encrypt research creator name", nil)
		return
	}

	response := EConsentFormResponse{
		ID:         updatedForm.ID,
		ResearchID: updatedForm.ResearchID,
		Research: ResearchResponseForConsent{
			ID:          updatedForm.Research.ID,
			Title:       updatedForm.Research.Title,
			Description: updatedForm.Research.Description,
			CategoryID:  updatedForm.Research.CategoryID,
			IsPublished: updatedForm.Research.IsPublished,
			UserName:    encryptedUserName,
		},
		Title:             updatedForm.Title,
		VideoID:           updatedForm.VideoID,
		VideoDescription:  updatedForm.VideoDescription,
		PersonalInfoCheck: updatedForm.PersonalInfoCheck,
		Coordinators:      coordinators,
		PatientAssigned:   patients,
		ExpiryTime:        updatedForm.ExpiryTime.UTC().Format(time.RFC3339),
		Video: VideoResponseForConsent{
			ID:          updatedForm.Video.ID,
			Title:       updatedForm.Video.Title,
			Description: updatedForm.Video.Description,
			FilePath:    updatedForm.Video.FilePath,
		},
		IsPublished: updatedForm.IsPublished,
		CreatedAt:   updatedForm.CreatedAt,
		UpdatedAt:   updatedForm.UpdatedAt,
	}

	// Add declarations to response if any exist
	if len(updatedForm.Declarations) > 0 {
		declarations := make([]ConsentDeclarationResponse, len(updatedForm.Declarations))
		for i, decl := range updatedForm.Declarations {
			declarations[i] = ConsentDeclarationResponse{
				ID:    decl.Declaration.ID,
				Label: decl.Declaration.Title,
			}
		}
		response.Declarations = declarations
	}

	// Add declaration_forms to the response
	declarationForms := make([]map[string]interface{}, 0, len(updatedForm.DeclarationForms))
	for _, df := range updatedForm.DeclarationForms {
		declarationForms = append(declarationForms, map[string]interface{}{
			"id":    df.ID,
			"label": df.Name,
		})
	}
	response.DeclarationForms = declarationForms

	utils.JSONResponse(c, http.StatusOK, "E-consent form updated successfully", response)
}

// PublishEConsentForm publishes an e-consent form (sets state to published and IsPublished to true)
func PublishEConsentForm(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Only admin or super-admin can publish
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can publish e-consent forms", nil)
		return
	}

	formIDStr := c.Param("id")
	if formIDStr == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "E-consent form ID is required", nil)
		return
	}
	formID, err := strconv.ParseUint(formIDStr, 10, 32)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid e-consent form ID", nil)
		return
	}

	var form models.EConsentForm
	if err := config.DB.Where("id = ? AND is_deleted = ?", formID, false).First(&form).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusNotFound, "E-consent form not found", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch e-consent form", nil)
		}
		return
	}

	if form.IsPublished {
		utils.JSONResponse(c, http.StatusConflict, "E-consent form is already published", nil)
		return
	}

	// --- Begin Transaction Block ---
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Gather declaration forms and their statements for the E-Consent
	var declarationForms []models.DeclarationForm
	if err := tx.Preload("Declarations").Model(&models.DeclarationForm{}).Joins("JOIN econsent_form_declaration_forms ON declaration_forms.id = econsent_form_declaration_forms.declaration_form_id").Where("econsent_form_declaration_forms.e_consent_form_id = ?", form.ID).Find(&declarationForms).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to load declaration forms", nil)
		return
	}

	type DeclarationSnapshot struct {
		ID           uint     `json:"id"`
		Title        string   `json:"title"`
		Statement    string   `json:"statement"`
		Options      []string `json:"options"`
		ResponseType string   `json:"response_type"`
	}

	type DeclarationFormSnapshot struct {
		ID          uint                 `json:"id"`
		Name        string               `json:"name"`
		Description string               `json:"description"`
		Statements  []DeclarationSnapshot `json:"statements"`
	}

	type ResearchDocumentSnapshot struct {
		ID       uint   `json:"id"`
		FileName string `json:"file_name"`
		FileType string `json:"file_type"`
	}

	type ResearchSnapshot struct {
		ID          uint                   `json:"id"`
		Title       string                 `json:"title"`
		Description string                 `json:"description"`
		CategoryID  uint                   `json:"category_id"`
		IsPublished bool                   `json:"is_published"`
		MimeType    string                 `json:"mimeType"`
		Document    *ResearchDocumentSnapshot `json:"document,omitempty"`
	}

	type VideoSnapshot struct {
		ID          uint   `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		FilePath    string `json:"file_path"`
		MimeType    string `json:"mimeType"`
	}

	type EConsentFormVersionSnapshot struct {
		Research         ResearchSnapshot           `json:"research"`
		Video            VideoSnapshot              `json:"video"`
		VideoDescription string                     `json:"video_description"`
		DeclarationForms []DeclarationFormSnapshot  `json:"declaration_forms"`
	}

	// Load research with documents
	var research models.Research
	if err := tx.Preload("Documents").Where("id = ?", form.ResearchID).First(&research).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to load research", nil)
		return
	}

	// Prepare research document snapshot (first document if exists)
	var researchDocSnap *ResearchDocumentSnapshot
	if len(research.Documents) > 0 {
		doc := research.Documents[0]
		researchDocSnap = &ResearchDocumentSnapshot{
			ID:       doc.ID,
			FileName: filepath.Base(doc.FilePath),
			FileType: doc.FileType,
		}
	}

	// Load video
	var video models.Video
	if err := tx.Where("id = ?", form.VideoID).First(&video).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to load video", nil)
		return
	}

	// Build the snapshot
	var formsSnapshot []DeclarationFormSnapshot
	for _, df := range declarationForms {
		log.Printf("DeclarationForm %d Declarations: %+v", df.ID, df.Declarations)
		var statements []DeclarationSnapshot
		for _, decl := range df.Declarations {
			statements = append(statements, DeclarationSnapshot{
				ID:    decl.ID,
				Title: decl.Title,
				Statement:  decl.Statement,
				Options: decl.Options,
				ResponseType: string(decl.ResponseType),
			})
		}
		formsSnapshot = append(formsSnapshot, DeclarationFormSnapshot{
			ID:          df.ID,
			Name:        df.Name,
			Description: df.Description,
			Statements:  statements, // always an array
		})
	}

	snapshot := EConsentFormVersionSnapshot{
		Research: ResearchSnapshot{
			ID:          research.ID,
			Title:       research.Title,
			Description: research.Description,
			CategoryID:  research.CategoryID,
			IsPublished: research.IsPublished,
			MimeType:    "application/pdf", // Default, or fetch from document if needed
			Document:    researchDocSnap,
		},
		Video: VideoSnapshot{
			ID:          video.ID,
			Title:       video.Title,
			Description: video.Description,
			FilePath:    video.FilePath,
			MimeType:    video.MimeType,
		},
		VideoDescription: form.VideoDescription,
		DeclarationForms: formsSnapshot,
	}

	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to serialize e-consent version snapshot", nil)
		return
	}

	// Find latest version number
	var latestVersion int
	var versionCount int64
	// First check if any versions exist
	if err := tx.Model(&models.EConsentFormVersion{}).
		Where("e_consent_form_id = ?", form.ID).
		Count(&versionCount).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to check existing versions", nil)
		return
	}
	
	if versionCount > 0 {
		// If versions exist, get the latest version number
		if err := tx.Model(&models.EConsentFormVersion{}).
			Where("e_consent_form_id = ?", form.ID).
			Select("MAX(version_number)").
			Scan(&latestVersion).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to get latest version number", nil)
			return
		}
	} else {
		// If no versions exist, start with version 1
		latestVersion = 0
	}
	
	// Always create a new version number to ensure proper versioning
	newVersionNumber := latestVersion + 1

	// Create new version
	version := models.EConsentFormVersion{
		EConsentFormID: form.ID,
		VersionNumber:  newVersionNumber,
		SnapshotJSON:   string(snapshotJSON),
		PublishedAt:    time.Now(),
	}
	
	log.Printf("Creating new version: FormID=%d, PreviousVersion=%d, NewVersionNumber=%d", form.ID, latestVersion, version.VersionNumber)
	
	if err := tx.Create(&version).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to create e-consent version: %v", err)
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save e-consent version", nil)
		return
	}
	
	log.Printf("Successfully created version with ID: %d", version.ID)

	// Update IsPublished inside the transaction
	updates := map[string]interface{}{
		"is_published": true,
		"updated_at":   time.Now(),
	}
	if err := tx.Model(&form).Updates(updates).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to publish e-consent form", nil)
		return
	}

	// Create EConsentAssignment entries for each assigned patient inside the transaction
	for _, patientID := range form.PatientAssigned {
		assignment := models.EConsentAssignment{
			EConsentFormID:        form.ID,
			PatientID:             patientID,
			AssignedAt:            time.Now(),
			Status:                models.EConsentStatusAssigned,
			IsDeleted:             false,
			EConsentFormVersionID: version.ID, // Link to version
		}
		if err := tx.Create(&assignment).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create e-consent assignment for patient", nil)
			return
		}
	}

	// Commit the transaction if all succeeded
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
		return
	}
	// --- End Transaction Block ---

	// Fetch updated form with relationships (use main DB, not tx)
	var updatedForm models.EConsentForm
	if err := config.DB.Preload("Research.CreatedByUser").
		Preload("Video").
		Where("id = ?", formID).
		First(&updatedForm).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch updated e-consent form", nil)
		return
	}

	// Add a debug log before calling extractPatientIDs
	log.Printf("DEBUG: updatedForm.PatientAssigned = %#v (type: %T)", updatedForm.PatientAssigned, updatedForm.PatientAssigned)
	patientIDs, err := extractPatientIDs(updatedForm.PatientAssigned)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to parse patient IDs", nil)
		return
	}

	// Use patientIDs for DB queries and getUsersInfo
	var patientUsers []models.User
	if err := config.DB.Where("id IN (?)", patientIDs).Find(&patientUsers).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch patients", nil)
		return
	}

	// Send notification to each assigned patient
	for _, patient := range patientUsers {
		title := "New E-Consent Assigned"
		message := "You have been assigned a new e-consent form: '" + updatedForm.Title + "' for research: '" + updatedForm.Research.Title + "'. Please review and submit your consent."
		notificationType := "E-Consent-Assign"
		entityType := "econsents"
		entityID := updatedForm.ID
		username := patient.FirstName + " " + patient.LastName
		metadata := map[string]interface{}{
			"econsent_form_id": updatedForm.ID,
			"research_id":      updatedForm.ResearchID,
			"status":           "assigned",
		}
		err := services.CreateNotification(config.DB, patient.ID, title, message, notificationType, username, entityType, entityID, metadata)
		if err != nil {
			// Log but do not fail the main response
			log.Printf("Failed to send notification to patient %d: %v", patient.ID, err)
		}

		// Send email notification
		emailBody := emails.EConsentAssignedEmail(
			patient.FirstName+" "+patient.LastName,
			updatedForm.Title,
			updatedForm.Research.Title,
		)
		subject := "New E-Consent Assigned"
		err = config.SendEmail([]string{patient.Email}, subject, emailBody)
		if err != nil {
			log.Printf("Failed to send email to patient %d: %v", patient.ID, err)
		}
	}

	// Prepare patients for response
	patients, err := getUsersInfo(patientIDs)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch patients", nil)
		return
	}

	// Get coordinator info
	coordinators, err := getUsersInfo(updatedForm.CoordinatorIDs)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch coordinators", nil)
		return
	}

	// Prepare response
	// Encrypt the research creator's username before including in the response
	creatorName := updatedForm.Research.CreatedByUser.FirstName + " " + updatedForm.Research.CreatedByUser.LastName
	encryptedUserName, err := utils.Encrypt(creatorName)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to encrypt research creator name", nil)
		return
	}

	response := EConsentFormResponse{
		ID:         updatedForm.ID,
		ResearchID: updatedForm.ResearchID,
		Research: ResearchResponseForConsent{
			ID:          updatedForm.Research.ID,
			Title:       updatedForm.Research.Title,
			Description: updatedForm.Research.Description,
			CategoryID:  updatedForm.Research.CategoryID,
			IsPublished: updatedForm.Research.IsPublished,
			UserName:    encryptedUserName,
		},
		Title:             updatedForm.Title,
		VideoID:           updatedForm.VideoID,
		VideoDescription:  updatedForm.VideoDescription,
		PersonalInfoCheck: updatedForm.PersonalInfoCheck,
		Coordinators:      coordinators,
		PatientAssigned:   patients,
		ExpiryTime:        updatedForm.ExpiryTime.UTC().Format(time.RFC3339),
		Video: VideoResponseForConsent{
			ID:          updatedForm.Video.ID,
			Title:       updatedForm.Video.Title,
			Description: updatedForm.Video.Description,
			FilePath:    updatedForm.Video.FilePath,
		},
		IsPublished: updatedForm.IsPublished,
		CreatedAt:   updatedForm.CreatedAt,
		UpdatedAt:   updatedForm.UpdatedAt,
	}

	// Add declarations to response if any exist
	if len(updatedForm.Declarations) > 0 {
		declarations := make([]ConsentDeclarationResponse, len(updatedForm.Declarations))
		for i, decl := range updatedForm.Declarations {
			declarations[i] = ConsentDeclarationResponse{
				ID:    decl.Declaration.ID,
				Label: decl.Declaration.Title,
			}
		}
		response.Declarations = declarations
	}

	utils.JSONResponse(c, http.StatusOK, "E-consent form published successfully", response)
}
