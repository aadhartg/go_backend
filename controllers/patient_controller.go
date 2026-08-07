package controllers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	// "strconv"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

// PatientEConsentSubmitRequest represents the request body for patient e-consent submission
type DeclarationResponseInput struct {
	DeclarationID uint        `json:"declaration_id" binding:"required"`
	Response      interface{} `json:"response"`
}

type FormDeclarationInput struct {
	FormID      uint                       `json:"form_id" binding:"required"`
	Declaration []DeclarationResponseInput `json:"declaration" binding:"required,dive,required"`
}

type PatientEConsentSubmitRequest struct {
	EConsentFormID       uint                   `json:"econsent_form_id" binding:"required"`
	NHI                  string                 `json:"nhi" binding:"required"`
	FirstName            string                 `json:"first_name" binding:"required"`
	LastName             string                 `json:"last_name" binding:"required"`
	Gender               string                 `json:"gender" binding:"required"`
	DOB                  string                 `json:"dob" binding:"required"`
	PatientSignURL       string                 `json:"patient_sign_url" binding:"required"`
	DeclarationResponses []FormDeclarationInput `json:"declaration_responses" binding:"required,dive,required"`
}

// PatientMyEConsentsResponse represents a single econsent assignment in the patient's view
type PatientMyEConsentsResponse struct {
	AssignmentID      uint       `json:"assignment_id"`
	EConsentFormID    uint       `json:"econsent_form_id"`
	EConsentTitle     string     `json:"econsent_title"`
	ResearchTitle     string     `json:"research_title"`
	ResearchID        uint       `json:"research_id"`
	AssignedAt        time.Time  `json:"assigned_at"`
	ExpiryTime        time.Time  `json:"expiry_time"`
	Status            string     `json:"status"`
	IsExpired         bool       `json:"is_expired"`
	CanSubmit         bool       `json:"can_submit"`
	PatientEConsentID *uint      `json:"patient_econsent_id,omitempty"`
	SubmittedAt       *time.Time `json:"submitted_at,omitempty"`
	ApprovedAt        *time.Time `json:"approved_at,omitempty"`
	VideoDescription  string     `json:"video_description"`
	PersonalInfoCheck bool       `json:"personal_info_check"`
	WithdrawalReason  string     `json:"withdrawal_reason,omitempty"`
	VersionNumber     string     `json:"version_number,omitempty"`
	IsLatestActive    bool       `json:"is_latest_active"` // Flag to identify the latest active consent for withdrawal
}

// PatientMyEConsentsListResponse represents the paginated response for patient's econsents
type PatientMyEConsentsListResponse struct {
	Page         int                          `json:"page"`
	PerPage      int                          `json:"per_page"`
	Sort         string                       `json:"sort"`
	SortColumn   string                       `json:"sort_column"`
	SearchText   string                       `json:"search_text"`
	Status       string                       `json:"status"`
	TotalRecords int64                        `json:"total_records"`
	TotalPages   int                          `json:"total_pages"`
	Records      []PatientMyEConsentsResponse `json:"records"`
}

// CreatePatientEConsent handles POST /patient-econsent to create a PatientEConsent entry
func CreatePatientEConsent(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, "Only authenticated patients can submit e-consent", nil)
		return
	}
	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Parse form fields
	eConsentFormIDStr := c.PostForm("econsent_form_id")
	eConsentFormID, err := strconv.ParseUint(eConsentFormIDStr, 10, 64)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid econsent_form_id", nil)
		return
	}
	nhi := c.PostForm("nhi")
	firstName := c.PostForm("first_name")
	lastName := c.PostForm("last_name")
	gender := c.PostForm("gender")
	dobStr := c.PostForm("dob")
	declarationResponsesStr := c.PostForm("declaration_responses")

	patientIDStr := c.PostForm("patient_id")

	var patientID uint

	if patientIDStr != "" {
		parsedID, err := strconv.ParseUint(patientIDStr, 10, 64)

		if err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid patient_id", nil)
			return
		}

		patientID = uint(parsedID)
	} else {
		// Self submit from patient portal
		patientID = user.ID
	}

	// Validate required fields
	missingFields := []string{}
	if nhi == "" {
		missingFields = append(missingFields, "NHI")
	}
	if firstName == "" {
		missingFields = append(missingFields, "First Name")
	}
	if lastName == "" {
		missingFields = append(missingFields, "Last Name")
	}
	if gender == "" {
		missingFields = append(missingFields, "Gender")
	}
	if dobStr == "" {
		missingFields = append(missingFields, "Dob")
	}
	if declarationResponsesStr == "" {
		missingFields = append(missingFields, "declarations")
	}
	if len(missingFields) > 0 {
		tx.Rollback()
		msg := "Missing required fields: " + strings.Join(missingFields, ", ")
		utils.JSONResponse(c, http.StatusBadRequest, msg, nil)
		return
	}

	// Decrypt and validate fields
	decryptedNHI, err := utils.Decrypt(nhi)
	if err != nil || decryptedNHI == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted NHI", nil)
		return
	}
	decryptedFirstName, err := utils.Decrypt(firstName)
	if err != nil || decryptedFirstName == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted first name", nil)
		return
	}
	decryptedLastName, err := utils.Decrypt(lastName)
	if err != nil || decryptedLastName == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted last name", nil)
		return
	}
	decryptedGender, err := utils.Decrypt(gender)
	if err != nil || decryptedGender == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted gender", nil)
		return
	}
	decryptedDOB, err := utils.Decrypt(dobStr)
	if err != nil || decryptedDOB == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted DOB", nil)
		return
	}

	// Parse new declaration_responses format (array of FormDeclarationInput)
	var declarationResponsesRaw []map[string]interface{}
	if err := json.Unmarshal([]byte(declarationResponsesStr), &declarationResponsesRaw); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid declaration_responses JSON", nil)
		return
	}
	if len(declarationResponsesRaw) == 0 {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "At least one declaration response is required", nil)
		return
	}

	var formDeclarationInputs []FormDeclarationInput
	for _, formItem := range declarationResponsesRaw {
		formIDRaw, okFormID := formItem["form_id"].(float64)
		declArr, okDeclArr := formItem["declaration"].([]interface{})
		if !okFormID || !okDeclArr {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid form declaration format", nil)
			return
		}
		formID := uint(formIDRaw)
		var declarations []DeclarationResponseInput
		for _, declItem := range declArr {
			declMap, ok := declItem.(map[string]interface{})
			if !ok {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid declaration item format", nil)
				return
			}
			declIDRaw, okID := declMap["declaration_id"].(float64)
			respRaw, okResp := declMap["response"]
			if !okID || !okResp {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid declaration response format", nil)
				return
			}
			declIDUint := uint(declIDRaw)
			// If response is string, decrypt; if bool, keep as is
			var respVal interface{}
			switch v := respRaw.(type) {
			case string:
				respDecrypted, err := utils.Decrypt(v)
				if err != nil || respDecrypted == "" {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted response", nil)
					return
				}
				// Try to parse as bool, else keep as string
				if respDecrypted == "true" || respDecrypted == "1" {
					respVal = true
				} else if respDecrypted == "false" || respDecrypted == "0" {
					respVal = false
				} else {
					respVal = respDecrypted
				}
			case bool:
				respVal = v
			default:
				tx.Rollback()
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid response type", nil)
				return
			}
			declarations = append(declarations, DeclarationResponseInput{
				DeclarationID: declIDUint,
				Response:      respVal,
			})
		}
		formDeclarationInputs = append(formDeclarationInputs, FormDeclarationInput{
			FormID:      formID,
			Declaration: declarations,
		})
	}

	// Convert []FormDeclarationInput to []services.FormDeclarationInput
	var serviceFormDeclarations []services.FormDeclarationInput
	for _, formInput := range formDeclarationInputs {
		var serviceDecls []services.DeclarationResponseInput
		for _, dr := range formInput.Declaration {
			serviceDecls = append(serviceDecls, services.DeclarationResponseInput{
				DeclarationID: dr.DeclarationID,
				Response:      dr.Response, // pass as-is (string or bool)
			})
		}
		serviceFormDeclarations = append(serviceFormDeclarations, services.FormDeclarationInput{
			FormID:      formInput.FormID,
			Declaration: serviceDecls,
		})
	}

	// Get the latest version for the form
	var latestVersion models.EConsentFormVersion
	err = config.DB.Where("e_consent_form_id = ?", uint(eConsentFormID)).Order("version_number DESC").First(&latestVersion).Error
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusNotFound, "E-consent version not found", nil)
		return
	}

	// Find the assignment for this patient, form, and latest version
	var assignment models.EConsentAssignment
	err = config.DB.Where("e_consent_form_id = ? AND patient_id = ? AND e_consent_form_version_id = ? AND is_deleted = ?", uint(eConsentFormID), patientID, latestVersion.ID, false).First(&assignment).Error
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "No e-consent assignment found for this patient and version", nil)
		return
	}
	// return
	// Use the latest version ID for the rest of the logic
	eConsentFormVersionID := latestVersion.ID

	// Call the service with the new format
	err = services.CreatePatientEConsentService(
		tx, user, patientID, user.ID, uint(eConsentFormID), eConsentFormVersionID, decryptedNHI, decryptedFirstName, decryptedLastName, decryptedGender, decryptedDOB, serviceFormDeclarations, c,
	)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// Notify all coordinators (nurses) that patient has signed
	var eConsentForm models.EConsentForm
	if err := config.DB.Preload("Research").Where("id = ?", uint(eConsentFormID)).First(&eConsentForm).Error; err == nil {
		for _, coordinatorID := range eConsentForm.CoordinatorIDs {
			// Notification
			_ = services.CreateNotification(
				config.DB,
				coordinatorID,
				"Patient Signed E-Consent",
				fmt.Sprintf("Patient %s %s has signed their e-consent form.", decryptedFirstName, decryptedLastName),
				"E-Consent",
				decryptedFirstName+" "+decryptedLastName,
				"econsents",
				uint(eConsentFormID),
				map[string]interface{}{
					"econsent_id": uint(eConsentFormID),
					"status":      "signed",
				},
			)

			// Email
			var coordinator models.User
			if err := config.DB.Where("id = ?", coordinatorID).First(&coordinator).Error; err == nil && coordinator.Email != "" {
				emailBody := fmt.Sprintf(
					"Dear %s,\n\nPatient %s %s has signed the e-consent form \"%s\" for research \"%s\".\n\nBest regards,\nTheranostics Team",
					coordinator.FirstName,
					decryptedFirstName,
					decryptedLastName,
					eConsentForm.Title,
					eConsentForm.Research.Title,
				)
				subject := "Patient Signed E-Consent Notification"
				_ = config.SendEmail([]string{coordinator.Email}, subject, emailBody)
			}
		}
	}

	utils.JSONResponse(c, http.StatusOK, "Patient e-consent created successfully", nil)
}

// GetPatientMyEConsents handles GET /api/patient/my-econsents to show patient's assigned econsents
func GetPatientMyEConsents(c *gin.Context) {
	// Get the user from the request context (set by authentication middleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Only allow patients to access this endpoint
	if user.RoleID != 4 {
		utils.JSONResponse(c, http.StatusForbidden, "Only patients can view their assigned econsents", nil)
		return
	}

	// Define allowed query parameters for filtering and pagination
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "status"}
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidQueryParameters, nil)
		return
	}

	// Parse and validate 'page' parameter (default: 1)
	page := 1
	if val := query.Get("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPageParameter, nil)
			return
		}
	}

	// Parse and validate 'per_page' parameter (default: 10)
	perPage := 10
	if val := query.Get("per_page"); val != "" {
		if pp, err := strconv.Atoi(val); err == nil && pp > 0 {
			perPage = pp
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPerPageParameter, nil)
			return
		}
	}

	// Parse and validate 'sort' parameter (default: desc)
	sort := "desc"
	if val := query.Get("sort"); val == "asc" || val == "desc" {
		sort = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortParameter, nil)
		return
	}

	// Parse and validate 'sort_column' parameter (default: assigned_at)
	sortColumn := "assigned_at"
	validSortColumns := []string{"econsent_title", "research_title", "status", "assigned_at", "expiry_time"}
	if val := query.Get("sort_column"); val != "" {
		if utils.StringInSlice(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortColumnParameter, nil)
			return
		}
	}

	// Map sortColumn to actual DB columns or expressions
	var sortExpr string
	switch sortColumn {
	case "econsent_title":
		sortExpr = "e_consent_forms.title " + sort
	case "research_title":
		sortExpr = "researches.title " + sort
	case "status":
		sortExpr = "e_consent_assignments.status " + sort
	case "expiry_time":
		sortExpr = "e_consent_forms.expiry_time " + sort
	case "assigned_at":
		fallthrough
	default:
		sortExpr = "e_consent_assignments.assigned_at " + sort
	}

	// Get the search text for filtering (optional)
	searchText := query.Get("search_text")

	// Get the status filter (optional)
	status := query.Get("status")
	validStatuses := map[string]bool{"submitted": true, "approved": true, "assigned": true, "in_progress": true, "expired": true, "withdrawn": true, "reviewed": true, "archived": true, "all": true,"expiring_soon": true, "": true}
	if !validStatuses[status] {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid status filter. Allowed values: all, assigned, submitted, approved, in progress and archived", nil)
		return
	}

	// Build the base GORM query for EConsentAssignment, joining with related tables
	db := config.DB.Model(&models.EConsentAssignment{}).
		Where("e_consent_assignments.patient_id = ? AND e_consent_assignments.is_deleted = ?", user.ID, false).
		Joins("LEFT JOIN e_consent_forms ON e_consent_assignments.e_consent_form_id = e_consent_forms.id").
		Joins("LEFT JOIN researches ON e_consent_forms.research_id = researches.id")

	// Apply status filter if not 'all' (or empty)
	if status != "all" && status != "" {
		db = db.Where("e_consent_assignments.status = ?", status)
	}

	// If search text is provided, filter by econsent title or research title
	if searchText != "" {
		searchPattern := "%" + searchText + "%"
		db = db.Where(
			"e_consent_forms.title ILIKE ? OR researches.title ILIKE ?",
			searchPattern, searchPattern,
		)
	}

	// Count total records for pagination
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

	// Apply sorting and pagination
	db = db.Order(sortExpr)
	offset := (page - 1) * perPage
	db = db.Limit(perPage).Offset(offset)

	// Fetch the assignments with related data
	var assignments []models.EConsentAssignment
	if err := db.Preload("EConsentForm.Research").
		Preload("EConsentForm.Video").
		Preload("EConsentFormVersion").
		Find(&assignments).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Only include assignments with status 'submitted', 'in_progress', 'assigned' or 'expired'
	filteredAssignments := make([]models.EConsentAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment.Status == models.EConsentStatusSubmitted ||
			assignment.Status == models.EConsentStatusInProgress ||
			assignment.Status == models.EConsentStatusExpired ||
			assignment.Status == models.EConsentStatusApproved ||
			assignment.Status == models.EConsentStatusAssigned ||
			assignment.Status == models.EConsentStatusWithdrawn ||
			assignment.Status == models.EConsentStatusReviewed ||
			assignment.Status == models.EConsentStatusArchived {
			filteredAssignments = append(filteredAssignments, assignment)
		}
	}

	// Prepare the response records
	var responses []PatientMyEConsentsResponse
	now := time.Now()

	for _, assignment := range filteredAssignments {
		// isExpired := assignment.EConsentForm.ExpiryTime.Before(now)
		// // Only expire if status is assigned or in_progress
		// if isExpired && (assignment.Status == models.EConsentStatusAssigned || assignment.Status == models.EConsentStatusInProgress) {
		// 	config.DB.Model(&models.EConsentAssignment{}).Where("id = ?", assignment.ID).Update("status", models.EConsentStatusExpired)
		// 	assignment.Status = models.EConsentStatusExpired // update local value too
		// }
		isExpired := assignment.EConsentForm.ExpiryTime.Before(now)

		if isExpired &&
			(assignment.Status == models.EConsentStatusAssigned ||
				assignment.Status == models.EConsentStatusInProgress) {

			config.DB.Model(&models.EConsentAssignment{}).
				Where("id = ?", assignment.ID).
				Update("status", models.EConsentStatusExpired)

			assignment.Status = models.EConsentStatusExpired

		} else if !isExpired &&
			assignment.Status == models.EConsentStatusExpired {

			config.DB.Model(&models.EConsentAssignment{}).
				Where("id = ?", assignment.ID).
				Update("status", models.EConsentStatusAssigned)

			assignment.Status = models.EConsentStatusAssigned
		}
		canSubmit := !isExpired && assignment.Status == models.EConsentStatusAssigned

		// Get patient econsent details if exists
		var patientEConsentID *uint
		var submittedAt *time.Time
		var approvedAt *time.Time

		var pe models.PatientEConsent
		// Always try to find a PatientEConsent record for the assignment.
		// It will exist for statuses like 'submitted', 'in progress', and 'approved'.
		if err := config.DB.Where("patient_id = ? AND e_consent_form_id = ? AND is_deleted = ?",
			user.ID, assignment.EConsentFormID, false).Order("created_at desc").First(&pe).Error; err == nil {
			patientEConsentID = &pe.ID
			submittedAt = pe.PatientSignedAt
			approvedAt = pe.NurseSignedAt
		}

		var withdrawalReason string
		if assignment.Status == models.EConsentStatusWithdrawn {
			withdrawalReason = assignment.WithdrawalReason
		}

		// Get version number and video description from version snapshot if available
		var versionNumber string
		var videoDescription string
		if assignment.EConsentFormVersionID != 0 {
			var version models.EConsentFormVersion
			if err := config.DB.Where("id = ?", assignment.EConsentFormVersionID).First(&version).Error; err == nil {
				versionNumber = strconv.Itoa(version.VersionNumber)

				// Try to get video description from version snapshot
				var snapshot struct {
					VideoDescription string `json:"video_description"`
				}
				if err := json.Unmarshal([]byte(version.SnapshotJSON), &snapshot); err == nil && snapshot.VideoDescription != "" {
					videoDescription = snapshot.VideoDescription
				} else {
					// Fallback to main form if snapshot doesn't have video description
					videoDescription = assignment.EConsentForm.VideoDescription
				}
			}
		} else {
			// No version, use main form
			videoDescription = assignment.EConsentForm.VideoDescription
		}

		response := PatientMyEConsentsResponse{
			AssignmentID:      assignment.ID,
			EConsentFormID:    assignment.EConsentFormID,
			EConsentTitle:     assignment.EConsentForm.Title,
			ResearchTitle:     assignment.EConsentForm.Research.Title,
			ResearchID:        assignment.EConsentForm.ResearchID,
			AssignedAt:        assignment.AssignedAt,
			ExpiryTime:        assignment.EConsentForm.ExpiryTime,
			Status:            string(assignment.Status),
			IsExpired:         isExpired,
			CanSubmit:         canSubmit,
			PatientEConsentID: patientEConsentID,
			SubmittedAt:       submittedAt,
			ApprovedAt:        approvedAt,
			VideoDescription:  videoDescription,
			PersonalInfoCheck: assignment.EConsentForm.PersonalInfoCheck,
			WithdrawalReason:  withdrawalReason,
			VersionNumber:     versionNumber,
		}

		responses = append(responses, response)
	}

	// Determine the latest active consent for withdrawal
	// A consent is considered "latest active" if it's the most recent approved consent for each e-consent form
	latestActiveMap := make(map[uint]int) // econsent_form_id -> index of latest active consent

	for i, response := range responses {
		// Only consider approved consents as candidates for "latest active"
		if response.Status == string(models.EConsentStatusApproved) {
			if existingIndex, exists := latestActiveMap[response.EConsentFormID]; exists {
				// If we already have a latest active for this form, compare submission times
				if response.SubmittedAt != nil && responses[existingIndex].SubmittedAt != nil {
					if response.SubmittedAt.After(*responses[existingIndex].SubmittedAt) {
						latestActiveMap[response.EConsentFormID] = i
					}
				}
			} else {
				// First approved consent for this form
				latestActiveMap[response.EConsentFormID] = i
			}
		}
	}

	// Mark the latest active consents
	for _, index := range latestActiveMap {
		responses[index].IsLatestActive = true
	}

	// Prepare the paginated response structure
	response := PatientMyEConsentsListResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		Status:       status,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      responses,
	}

	utils.JSONResponse(c, http.StatusOK, "Patient econsents fetched successfully", response)
}

// PatientResignEConsent allows a patient to re-sign an existing e-consent after a resign request
func PatientResignEConsent(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok || user.RoleID != 4 {
		utils.JSONResponse(c, http.StatusUnauthorized, "Only authenticated patients can re-sign e-consent", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid patient_econsent_id", nil)
		return
	}

	// Find the PatientEConsent record
	var pe models.PatientEConsent
	if err := config.DB.Where("id = ? AND is_deleted = ?", id, false).First(&pe).Error; err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Patient e-consent not found", nil)
		return
	}

	// Check ownership
	if pe.PatientID != user.ID {
		utils.JSONResponse(c, http.StatusForbidden, "You are not authorized to re-sign this e-consent", nil)
		return
	}

	// Only allow if current status is 'in progress'
	if pe.Status != models.EConsentStatusInProgress {
		utils.JSONResponse(c, http.StatusBadRequest, "Can only re-sign if status is 'in_progress'", nil)
		return
	}

	// Fetch the EConsentForm to get the expiry time
	var eConsentForm models.EConsentForm
	if err := config.DB.Where("id = ?", pe.EConsentFormID).First(&eConsentForm).Error; err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "E-consent form not found", nil)
		return
	}
	// Cannot resign the consent if its expired
	expired, msg := services.CheckEConsentExpiry(config.DB, pe.EConsentFormID, &pe.ID, nil)
	if expired {
		utils.JSONResponse(c, http.StatusForbidden, msg, nil)
		return
	}

	// Handle signature upload
	var patientSignURL, patientSignHash string
	if file, err := c.FormFile("patient_sign_file"); err == nil && file != nil {
		hash, fileBytes, err := utils.ComputeFileHashFromFormFile(file)
		if err != nil {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to compute signature hash", nil)
			return
		}
		signDir := "public/signatures/patients"
		_ = os.MkdirAll(signDir, 0755)
		fileName := fmt.Sprintf("patient_%d_%s%s", user.ID, hash, filepath.Ext(file.Filename))
		filePath := filepath.Join(signDir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := os.WriteFile(filePath, fileBytes, 0644); err != nil {
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save new signature file", nil)
				return
			}
		}
		patientSignURL = "/signatures/patients/" + fileName
		patientSignHash = hash
		// Update user's default patient signature URL and hash
		user.DefaultPatientSignatureURL = patientSignURL
		user.DefaultPatientSignatureHash = patientSignHash
		_ = config.DB.Save(&user)
	} else {
		utils.JSONResponse(c, http.StatusBadRequest, "A new signature file is required.", nil)
		return
	}

	// Update PatientEConsent record
	pe.PatientSignURL = patientSignURL
	pe.PatientSignHash = patientSignHash
	pe.PatientSignedAt = utils.TimePtr(time.Now())
	pe.Status = models.EConsentStatusSubmitted
	pe.UpdatedAt = time.Now()
	if err := config.DB.Save(&pe).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update e-consent with new signature", nil)
		return
	}

	// Update assignment status if exists
	config.DB.Model(&models.EConsentAssignment{}).
		Where("e_consent_form_id = ? AND patient_id = ? AND is_deleted = ?", pe.EConsentFormID, user.ID, false).
		Updates(map[string]interface{}{"status": models.EConsentStatusSubmitted, "submitted_at": time.Now()})

	// Notify all coordinators (nurses) that patient has re-signed
	if err := config.DB.Where("id = ?", pe.EConsentFormID).First(&eConsentForm).Error; err == nil {
		for _, coordinatorID := range eConsentForm.CoordinatorIDs {
			_ = services.CreateNotification(
				config.DB,
				coordinatorID,
				"Patient Re-signed E-Consent",
				fmt.Sprintf("Patient %s %s has re-signed their e-consent form.", user.FirstName, user.LastName),
				"E-Consent",
				user.FirstName+" "+user.LastName,
				"econsents",
				pe.ID,
				map[string]interface{}{
					"econsent_id": pe.ID,
					"status":      "resigned",
				},
			)
		}
	}

	utils.JSONResponse(c, http.StatusOK, "E-consent re-signed and submitted successfully", map[string]interface{}{
		"patient_sign_url": patientSignURL,
	})
}

// GetPatientEConsentFormDetails handles GET /api/patient/econsent-form/:id to fetch econsent form details for a patient to start filling
func GetPatientEConsentFormDetails(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)

	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, "Only authenticated patients can access e-consent form details", nil)
		return
	}

	formIDStr := c.Param("id")
	patientIdStr := c.Param("patientId")

	formID, err := strconv.ParseUint(formIDStr, 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid econsent_form_id", nil)
		return
	}
	var patientId uint

	if patientIdStr != "" {

		id, err := strconv.ParseUint(patientIdStr, 10, 32)
		if err != nil {
			utils.JSONResponse(
				c,
				http.StatusBadRequest,
				"Invalid patient id",
				nil,
			)
			return
		}

		patientId = uint(id)

	} else {
		patientId = user.ID
	}

	// Fetch the EConsentAssignment for this patient and form
	var assignment models.EConsentAssignment
	err = config.DB.Where("e_consent_form_id = ? AND patient_id = ? AND is_deleted = ?", uint(formID), patientId, false).First(&assignment).Error
	if err != nil {
		utils.JSONResponse(c, http.StatusForbidden, "No e-consent assignment found for this patient", nil)
		return
	}
	var isVerified bool
	isVerified = assignment.IsVerified

	// Fetch the latest versioned snapshot for this e-consent form
	var version models.EConsentFormVersion
	err = config.DB.Where("e_consent_form_id = ?", assignment.EConsentFormID).
		Order("version_number DESC").
		First(&version).Error
	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "E-consent version not found", nil)
		return
	}

	// Unmarshal the snapshot JSON
	var snapshot struct {
		Research         map[string]interface{}   `json:"research"`
		Video            map[string]interface{}   `json:"video"`
		VideoDescription string                   `json:"video_description"`
		DeclarationForms []map[string]interface{} `json:"declaration_forms"`
	}
	if err := json.Unmarshal([]byte(version.SnapshotJSON), &snapshot); err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to parse e-consent version snapshot", nil)
		return
	}

	// Fetch the EConsentForm for non-versioned fields
	var form models.EConsentForm
	err = config.DB.Where("id = ? AND is_deleted = ?", uint(formID), false).First(&form).Error
	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "E-consent form not found", nil)
		return
	}

	// Try to find an existing PatientEConsent for this patient and form to prefill NHI and DOB
	var patientEConsent models.PatientEConsent
	err = config.DB.Where("patient_id = ? AND e_consent_form_id = ? AND e_consent_form_version_id = ? AND is_deleted = ?", patientId, form.ID, version.ID, false).Order("created_at desc").First(&patientEConsent).Error
	var firstName, lastName, gender, nhi, dob string
	if err == nil {
		firstName = patientEConsent.FirstName
		lastName = patientEConsent.LastName
		gender = patientEConsent.Gender
		nhi = patientEConsent.NHI
		dob = patientEConsent.DOB
	} else if form.PersonalInfoCheck {
		// Prefill from user model if personal_info_check is true and no PatientEConsent exists
		var patient models.User

		err := config.DB.
			Where("id = ?", patientId).
			First(&patient).Error

		if err != nil {
			utils.JSONResponse(
				c,
				http.StatusNotFound,
				"Patient not found",
				nil,
			)
			return
		}

		encFirstName, _ := utils.Encrypt(patient.FirstName)
		encLastName, _ := utils.Encrypt(patient.LastName)
		encNHI, _ := utils.Encrypt(patient.NHI)

		encDOB := ""

		if patient.DOB != nil {
			encDOB, _ = utils.Encrypt(
				patient.DOB.Format("2006-01-02"),
			)
		}
		encGender := patient.Gender
		firstName = encFirstName
		lastName = encLastName
		gender = encGender
		dob = encDOB
		nhi = encNHI
	}

	// Prepare research document info (return the first document if exists)
	var researchDoc map[string]interface{}
	if err := config.DB.Preload("Documents").First(&form.Research, form.ResearchID).Error; err == nil {
		if len(form.Research.Documents) > 0 {
			doc := form.Research.Documents[0]
			researchDoc = map[string]interface{}{
				"id":        doc.ID,
				"file_name": filepath.Base(doc.FilePath),
				"file_type": doc.FileType,
			}
		}
	}

	// Prepare video info for response (from versioned snapshot if available, else from live)
	var videoInfo map[string]interface{}
	if snapshot.Video != nil && len(snapshot.Video) > 0 {
		videoInfo = snapshot.Video
	} else if form.Video.ID != 0 {
		videoInfo = map[string]interface{}{
			"id":          form.Video.ID,
			"title":       form.Video.Title,
			"description": form.Video.Description,
			"file_path":   filepath.Base(form.Video.FilePath),
		}
	}

	// Transform declaration_forms to required format
	var formattedDeclarationForms []map[string]interface{}
	for _, form := range snapshot.DeclarationForms {
		statements := []map[string]interface{}{}
		// Try both "declarations" and "statements" keys for compatibility
		var decls []interface{}
		if d, ok := form["declarations"]; ok {
			decls, _ = d.([]interface{})
		} else if d, ok := form["statements"]; ok {
			decls, _ = d.([]interface{})
		}
		for _, d := range decls {
			if decl, ok := d.(map[string]interface{}); ok {
				statements = append(statements, map[string]interface{}{
					"id":            decl["id"],
					"title":         decl["title"],
					"statement":     decl["statement"],
					"response_type": decl["response_type"],
					"options":       decl["options"],
				})
			}
		}
		formattedDeclarationForms = append(formattedDeclarationForms, map[string]interface{}{
			"id":         form["id"],
			"label":      form["name"],
			"statements": statements,
		})
	}

	resp := map[string]interface{}{
		"econsent_form_id":         form.ID,
		"research_id":              form.ResearchID,
		"title":                    form.Title,
		"version_number":           version.VersionNumber,     // Add version number here
		"econsent_form_version_id": version.ID,                // Add version ID here
		"video_description":        snapshot.VideoDescription, // from version
		"personal_info_check":      form.PersonalInfoCheck,    // from live
		"expiry_time":              form.ExpiryTime,           // from live
		"declaration_forms":        formattedDeclarationForms, // formatted as required
		"research_document":        researchDoc,               // from live, without mimeType
		"video":                    videoInfo,                 // merged
		"user": map[string]interface{}{
			"first_name": firstName,
			"last_name":  lastName,
			"gender":     gender,
			"nhi":        nhi,
			"dob":        dob,
		},
		"isVerified":   isVerified,
		"assignmentId": assignment.ID,
	}

	utils.JSONResponse(c, http.StatusOK, "E-consent form details fetched successfully", resp)
}

// GetPatientAssignedEConsents returns assigned e-consents for the authenticated patient, with filtering and pagination like GetPatientMyEConsents
func GetPatientAssignedEConsents(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}
	if user.RoleID != 4 {
		utils.JSONResponse(c, http.StatusForbidden, "Only patients can view their assigned econsents", nil)
		return
	}

	// Allowed query params
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "status"}
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidQueryParameters, nil)
		return
	}

	// Parse and validate 'page' (default: 1)
	page := 1
	if val := query.Get("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPageParameter, nil)
			return
		}
	}

	// Parse and validate 'per_page' (default: 10)
	perPage := 10
	if val := query.Get("per_page"); val != "" {
		if pp, err := strconv.Atoi(val); err == nil && pp > 0 {
			perPage = pp
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPerPageParameter, nil)
			return
		}
	}

	// Parse and validate 'sort' (default: desc)
	sort := "desc"
	if val := query.Get("sort"); val == "asc" || val == "desc" {
		sort = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortParameter, nil)
		return
	}

	// Parse and validate 'sort_column' (default: assigned_at)
	sortColumn := "assigned_at"
	validSortColumns := []string{"econsent_title", "research_title", "status", "assigned_at", "expiry_time"}
	if val := query.Get("sort_column"); val != "" {
		if utils.StringInSlice(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortColumnParameter, nil)
			return
		}
	}

	// Map sortColumn to actual DB columns or expressions
	var sortExpr string
	switch sortColumn {
	case "econsent_title":
		sortExpr = "e_consent_forms.title " + sort
	case "research_title":
		sortExpr = "researches.title " + sort
	case "status":
		sortExpr = "e_consent_assignments.status " + sort
	case "expiry_time":
		sortExpr = "e_consent_forms.expiry_time " + sort
	case "assigned_at":
		fallthrough
	default:
		sortExpr = "e_consent_assignments.assigned_at " + sort
	}

	// Get the search text for filtering (optional)
	searchText := query.Get("search_text")

	// Get the status filter (default: assigned)
	status := query.Get("status")
	if status == "" {
		status = string(models.EConsentStatusAssigned)
	}
	validStatuses := map[string]bool{"assigned": true, "all": true, "": true}
	if !validStatuses[status] {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid status filter. Allowed values: all, assigned, submitted, approved, in progress.", nil)
		return
	}

	// Build the base GORM query for EConsentAssignment, joining with related tables
	db := config.DB.Model(&models.EConsentAssignment{}).
		Where("e_consent_assignments.patient_id = ? AND e_consent_assignments.is_deleted = ?", user.ID, false).
		Joins("LEFT JOIN e_consent_forms ON e_consent_assignments.e_consent_form_id = e_consent_forms.id").
		Joins("LEFT JOIN researches ON e_consent_forms.research_id = researches.id")

	// Apply status filter if not 'all' (or empty)
	if status != "all" && status != "" {
		db = db.Where("e_consent_assignments.status = ?", status)
	}

	// If search text is provided, filter by econsent title or research title
	if searchText != "" {
		searchPattern := "%" + searchText + "%"
		db = db.Where(
			"e_consent_forms.title ILIKE ? OR researches.title ILIKE ?",
			searchPattern, searchPattern,
		)
	}

	// Count total records for pagination
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

	// Apply sorting and pagination
	db = db.Order(sortExpr)
	offset := (page - 1) * perPage
	db = db.Limit(perPage).Offset(offset)

	// Fetch the assignments with related data
	var assignments []models.EConsentAssignment
	if err := db.Preload("EConsentForm.Research").
		Preload("EConsentForm.Video").
		Preload("EConsentFormVersion").
		Find(&assignments).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Prepare the response records
	var responses []PatientMyEConsentsResponse
	now := time.Now()

	for _, assignment := range assignments {
		isExpired := assignment.EConsentForm.ExpiryTime.Before(now)
		// Only expire if status is assigned or in_progress
		if isExpired && (assignment.Status == models.EConsentStatusAssigned || assignment.Status == models.EConsentStatusInProgress) {
			config.DB.Model(&assignment).Update("status", models.EConsentStatusExpired)
			assignment.Status = models.EConsentStatusExpired // update local value too
		}
		canSubmit := !isExpired && assignment.Status == models.EConsentStatusAssigned

		// Get patient econsent details if exists
		var patientEConsentID *uint
		var submittedAt *time.Time
		var approvedAt *time.Time

		var pe models.PatientEConsent
		if err := config.DB.Where("patient_id = ? AND e_consent_form_id = ? AND is_deleted = ?",
			user.ID, assignment.EConsentFormID, false).Order("created_at desc").First(&pe).Error; err == nil {
			patientEConsentID = &pe.ID
			submittedAt = pe.PatientSignedAt
			approvedAt = pe.NurseSignedAt
		}

		// Get version number if available
		var versionNumber string
		if assignment.EConsentFormVersionID != 0 {
			var version models.EConsentFormVersion
			if err := config.DB.Where("id = ?", assignment.EConsentFormVersionID).First(&version).Error; err == nil {
				versionNumber = strconv.Itoa(version.VersionNumber)
			}
		}

		response := PatientMyEConsentsResponse{
			AssignmentID:      assignment.ID,
			EConsentFormID:    assignment.EConsentFormID,
			EConsentTitle:     assignment.EConsentForm.Title,
			ResearchTitle:     assignment.EConsentForm.Research.Title,
			ResearchID:        assignment.EConsentForm.ResearchID,
			AssignedAt:        assignment.AssignedAt,
			ExpiryTime:        assignment.EConsentForm.ExpiryTime,
			Status:            string(assignment.Status),
			IsExpired:         isExpired,
			CanSubmit:         canSubmit,
			PatientEConsentID: patientEConsentID,
			SubmittedAt:       submittedAt,
			ApprovedAt:        approvedAt,
			VideoDescription:  assignment.EConsentForm.VideoDescription,
			PersonalInfoCheck: assignment.EConsentForm.PersonalInfoCheck,
			VersionNumber:     versionNumber,
		}

		responses = append(responses, response)
	}

	// Determine the latest active consent for withdrawal
	// A consent is considered "latest active" if it's the most recent approved consent for each e-consent form
	latestActiveMap := make(map[uint]int) // econsent_form_id -> index of latest active consent

	for i, response := range responses {
		// Only consider approved consents as candidates for "latest active"
		if response.Status == string(models.EConsentStatusApproved) {
			if existingIndex, exists := latestActiveMap[response.EConsentFormID]; exists {
				// If we already have a latest active for this form, compare submission times
				if response.SubmittedAt != nil && responses[existingIndex].SubmittedAt != nil {
					if response.SubmittedAt.After(*responses[existingIndex].SubmittedAt) {
						latestActiveMap[response.EConsentFormID] = i
					}
				}
			} else {
				// First approved consent for this form
				latestActiveMap[response.EConsentFormID] = i
			}
		}
	}

	// Mark the latest active consents
	for _, index := range latestActiveMap {
		responses[index].IsLatestActive = true
	}

	// Prepare the paginated response structure
	resp := PatientMyEConsentsListResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		Status:       status,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      responses,
	}

	utils.JSONResponse(c, http.StatusOK, "Assigned e-consents fetched successfully", resp)
}

// PatientWithdrawEConsent allows a patient to withdraw an approved e-consent
func PatientWithdrawEConsent(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok || user.RoleID != 4 {
		utils.JSONResponse(c, http.StatusUnauthorized, "Only authenticated patients can withdraw e-consent", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid patient_econsent_id", nil)
		return
	}

	// Parse withdrawal reason from JSON body
	var req struct {
		WithdrawalReason string `json:"withdrawal_reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.WithdrawalReason) == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "A withdrawal reason is required", nil)
		return
	}

	// Decrypt the withdrawal reason
	decryptedReason, err := utils.Decrypt(req.WithdrawalReason)
	if err != nil || strings.TrimSpace(decryptedReason) == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted withdrawal reason", nil)
		return
	}

	// Encrypt again before saving
	encryptedReason, err := utils.Encrypt(decryptedReason)
	if err != nil || strings.TrimSpace(encryptedReason) == "" {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to encrypt withdrawal reason", nil)
		return
	}

	// Find the PatientEConsent record
	var pe models.PatientEConsent
	if err := config.DB.Where("id = ? AND is_deleted = ?", id, false).First(&pe).Error; err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Patient e-consent not found", nil)
		return
	}

	// Check ownership
	if pe.PatientID != user.ID {
		utils.JSONResponse(c, http.StatusForbidden, "You are not authorized to withdraw this e-consent", nil)
		return
	}

	// Only allow if current status is 'approved'
	if pe.Status != models.EConsentStatusApproved {
		utils.JSONResponse(c, http.StatusBadRequest, "Can only withdraw if status is 'approved'", nil)
		return
	}

	// Start transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Update PatientEConsent status and reason
	pe.Status = models.EConsentStatusWithdrawn
	pe.WithdrawalReason = encryptedReason
	if err := tx.Save(&pe).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update e-consent status", nil)
		return
	}

	// Withdraw ALL approved consents for this patient and e-consent form (including previous versions)
	// This ensures that when a patient withdraws, all their approved consents for this form are withdrawn
	var allApprovedConsents []models.PatientEConsent
	if err := tx.Where("patient_id = ? AND e_consent_form_id = ? AND status = ? AND is_deleted = ?",
		pe.PatientID, pe.EConsentFormID, models.EConsentStatusApproved, false).Find(&allApprovedConsents).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch approved consents", nil)
		return
	}

	// Update all approved consents to withdrawn status
	for _, consent := range allApprovedConsents {
		consent.Status = models.EConsentStatusWithdrawn
		consent.WithdrawalReason = encryptedReason
		if err := tx.Save(&consent).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update consent status", nil)
			return
		}
	}

	// Update ALL approved EConsentAssignment records for this patient and e-consent form to withdrawn status
	// Only withdraw assignments that are approved, not archived or other statuses
	var allApprovedAssignments []models.EConsentAssignment
	if err := tx.Where("patient_id = ? AND e_consent_form_id = ? AND status = ? AND is_deleted = ?",
		pe.PatientID, pe.EConsentFormID, models.EConsentStatusApproved, false).Find(&allApprovedAssignments).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch approved assignments", nil)
		return
	}

	// Update all approved assignments to withdrawn status
	for _, assignment := range allApprovedAssignments {
		assignment.Status = models.EConsentStatusWithdrawn
		assignment.WithdrawalReason = encryptedReason
		if err := tx.Save(&assignment).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update assignment status", nil)
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
		return
	}

	// Notify all coordinators (nurses) that patient has withdrawn
	var eConsentForm models.EConsentForm
	if err := config.DB.Preload("Research").Where("id = ?", pe.EConsentFormID).First(&eConsentForm).Error; err == nil {
		for _, coordinatorID := range eConsentForm.CoordinatorIDs {
			_ = services.CreateNotification(
				config.DB,
				coordinatorID,
				"Patient Withdrew E-Consent",
				fmt.Sprintf("Patient %s %s has withdrawn their e-consent form and all previous approved versions. Reason: %s", user.FirstName, user.LastName, decryptedReason),
				"E-Consent",
				user.FirstName+" "+user.LastName,
				"econsents",
				pe.ID,
				map[string]interface{}{
					"econsent_id": pe.ID,
					"status":      "withdrawn",
				},
			)

			// Send email to coordinator
			var coordinator models.User
			if err := config.DB.Where("id = ?", coordinatorID).First(&coordinator).Error; err == nil && coordinator.Email != "" {
				emailBody := emails.CoordinatorConsentWithdrawnEmail(
					user.FirstName+" "+user.LastName,
					eConsentForm.Title,
					eConsentForm.Research.Title,
					decryptedReason,
				)
				subject := "Patient Consent Withdrawal Notification"
				_ = config.SendEmail([]string{coordinator.Email}, subject, emailBody)
			}
		}
	}

	utils.JSONResponse(c, http.StatusOK, "E-consent withdrawn successfully", nil)
}

func SendEConsentVerificationCode(c *gin.Context) {
	_, ok := middlewares.GetUserFromContext(c)

	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	idParam := c.Param("id")

	assignmentID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid assignment id", nil)
		return
	}

	// Fetch assignment
	var assignment models.EConsentAssignment

	err = config.DB.
		Preload("User").
		First(&assignment, uint(assignmentID)).Error

	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Assignment not found", nil)
		return
	}

	if assignment.User.Email == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Patient email not found", nil)
		return
	}

	// =========================
	// Rate Limiting
	// =========================

	cooldownKey := fmt.Sprintf("consent_verification_cooldown:%d", assignment.ID)
	limitKey := fmt.Sprintf("consent_verification_limit:%d", assignment.ID)

	// Cooldown check (30 sec)
	exists, err := config.RedisClient.Exists(config.Ctx, cooldownKey).Result()
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Rate limit check failed", nil)
		return
	}

	if exists > 0 {
		utils.JSONResponse(
			c,
			http.StatusTooManyRequests,
			"Please wait 30 seconds before requesting another verification code",
			nil,
		)
		return
	}

	// Increment request count
	count, err := config.RedisClient.Incr(config.Ctx, limitKey).Result()
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Rate limit check failed", nil)
		return
	}

	// Set 15-minute window on first request
	if count == 1 {
		config.RedisClient.Expire(config.Ctx, limitKey, 15*time.Minute)
	}

	// Max 5 requests in 15 minutes
	if count > 5 {
		utils.JSONResponse(
			c,
			http.StatusTooManyRequests,
			"Too many verification code requests. Please try again later.",
			nil,
		)
		return
	}

	// Start cooldown
	config.RedisClient.Set(
		config.Ctx,
		cooldownKey,
		"1",
		30*time.Second,
	)

	// =========================
	// Generate OTP
	// =========================

	otp := fmt.Sprintf("%06d", rand.Intn(1000000))

	otpKey := fmt.Sprintf("consent_verification:%d", assignment.ID)

	err = config.RedisClient.Set(
		config.Ctx,
		otpKey,
		otp,
		5*time.Minute,
	).Err()

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusInternalServerError,
			"Failed to store verification code",
			nil,
		)
		return
	}

	// =========================
	// Send Email
	// =========================

	emailBody := emails.VerificationCodeEmail(
		assignment.User.FirstName,
		assignment.User.LastName,
		otp,
	)

	if err := config.SendEmail(
		[]string{assignment.User.Email},
		"E-Consent Verification Code",
		emailBody,
	); err != nil {

		// Remove OTP if email fails
		config.RedisClient.Del(config.Ctx, otpKey)

		utils.JSONResponse(
			c,
			http.StatusInternalServerError,
			"Failed to send verification email",
			nil,
		)
		return
	}

	utils.JSONResponse(
		c,
		http.StatusOK,
		"Verification code sent successfully",
		nil,
	)
}

func VerifyEConsentVerificationCode(c *gin.Context) {
	_, ok := middlewares.GetUserFromContext(c)

	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	idParam := c.Param("id")

	assignmentID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid assignment id", nil)
		return
	}

	type VerifyRequest struct {
		Code string `json:"code"`
	}

	var req VerifyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	// Fetch assignment
	var assignment models.EConsentAssignment

	err = config.DB.
		Preload("User").
		First(&assignment, uint(assignmentID)).Error

	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Assignment not found", nil)
		return
	}

	// Redis Key
	key := fmt.Sprintf("consent_verification:%d", assignment.ID)

	// Fetch OTP from Redis
	storedOTP, err := config.RedisClient.Get(
		config.Ctx,
		key,
	).Result()

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusBadRequest,
			"Verification code expired or invalid",
			nil,
		)
		return
	}

	// Compare OTP
	if storedOTP != req.Code {
		utils.JSONResponse(
			c,
			http.StatusBadRequest,
			"Invalid verification code",
			nil,
		)
		return
	}

	// Mark assignment as verified
	assignment.IsVerified = true

	err = config.DB.Save(&assignment).Error

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusInternalServerError,
			"Failed to update verification status",
			nil,
		)
		return
	}

	// Delete OTP after successful verification
	config.RedisClient.Del(config.Ctx, key)

	utils.JSONResponse(
		c,
		http.StatusOK,
		"Verification successful",
		nil,
	)
}
