package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/dto"
	"theransticslabs/m/emails"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

// PatientConsentAssignmentResponse represents a single patient consent assignment in the response
type PatientConsentAssignmentResponse struct {
	AssignmentID   uint   `json:"assignment_id"`
	PatientID      uint   `json:"patient_id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	EConsentFormID uint   `json:"econsent_form_id"`
	EConsentTitle  string `json:"econsent_title"`
	AssignedAt     string `json:"assigned_at"`
	Status         string `json:"status"`
}

type PatientConsentAssignmentListResponse struct {
	Page         int                                `json:"page"`
	PerPage      int                                `json:"per_page"`
	Sort         string                             `json:"sort"`
	SortColumn   string                             `json:"sort_column"`
	SearchText   string                             `json:"search_text"`
	TotalRecords int64                              `json:"total_records"`
	TotalPages   int                                `json:"total_pages"`
	Records      []PatientConsentAssignmentResponse `json:"records"`
}

// GeneratePatientConsentPDFHandler handles GET /patient-management/:patient_econsent_id/pdf and returns a PDF file
type DeclarationPDFInfo struct {
	Statement string
	Response  bool
}

// ListPatientConsentAssignments lists all patients to whom a consent is assigned, with filtering
func ListPatientConsentAssignments(c *gin.Context) {
	// Get the user from the request context (set by authentication middleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		// If user is not authenticated, return unauthorized
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Only allow admin or super-admin to access this endpoint
	if user.RoleID != 1 && user.RoleID != 2 && user.RoleID != 5 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can view patient consent assignments", nil)
		return
	}

	// Define allowed query parameters for filtering and pagination
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "status"}
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		// If any unexpected query parameter is present, return bad request
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
	if val := strings.ToLower(query.Get("sort")); val == "asc" || val == "desc" {
		sort = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortParameter, nil)
		return
	}

	// Parse and validate 'sort_column' parameter (default: assigned_at)
	sortColumn := "assigned_at"
	validSortColumns := []string{"patient_name", "patient_email", "econsent_title", "status", "assigned_at", "expiry_time"}
	if val := strings.ToLower(query.Get("sort_column")); val != "" {
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
	case "patient_name":
		sortExpr = "users.first_name || ' ' || users.last_name " + sort
	case "patient_email":
		sortExpr = "users.email " + sort
	case "econsent_title":
		sortExpr = "e_consent_forms.title " + sort
	case "status":
		sortExpr = "e_consent_assignments.status " + sort
	case "assigned_at":
		fallthrough
	default:
		sortExpr = "e_consent_assignments.assigned_at " + sort
	}

	// Get the search text for filtering (optional)
	searchText := strings.TrimSpace(query.Get("search_text"))

	// Get the status filter (optional, default: assigned)
	status := strings.ToLower(strings.TrimSpace(query.Get("status")))
	validStatuses := map[string]bool{"assigned": true, "submitted": true, "approved": true, "in_progress": true, "expired": true, "all": true, "withdrawn": true, "archived": true, "reviewed": true, "expiring_soon": true, "": true}
	if !validStatuses[status] {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid status filter. Allowed values: all, assigned, submitted, approved, in progress, archived", nil)
		return
	}

	// Build the base GORM query for EConsentAssignment, joining users and e_consent_forms
	db := config.DB.Model(&models.EConsentAssignment{}).
		Where("e_consent_assignments.is_deleted = ?", false).
		Joins("LEFT JOIN users ON e_consent_assignments.patient_id = users.id").
		Joins("LEFT JOIN e_consent_forms ON e_consent_assignments.e_consent_form_id = e_consent_forms.id")

	// // Apply status filter if not 'all' (or empty)
	// if status != "all" && status != "" {
	// 	db = db.Where("e_consent_assignments.status = ?", status)
	// }
	if status == "expiring_soon" {

		twoDaysFromNow := time.Now().AddDate(0, 0, 2)

		db = db.Where(`
        e_consent_assignments.status = ?
        AND e_consent_forms.expiry_time <= ?
        AND e_consent_forms.expiry_time > ?
        AND e_consent_forms.is_deleted = ?
    `,
			"assigned",
			twoDaysFromNow,
			time.Now(),
			false,
		)

	} else if status != "all" && status != "" {

		db = db.Where("e_consent_assignments.status = ?", status)
	}

	// If search text is provided, filter by patient name, email, e-consent form title, or status
	if searchText != "" {
		searchPattern := "%" + searchText + "%"
		db = db.Where(
			"users.first_name ILIKE ? OR users.last_name ILIKE ? OR (users.first_name || ' ' || users.last_name) ILIKE ? OR users.email ILIKE ? OR e_consent_forms.title ILIKE ? OR e_consent_assignments.status ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern,
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

	// Fetch the assignments with related user and e-consent form data
	var assignments []models.EConsentAssignment
	if err := db.Preload("EConsentForm").Preload("User").Find(&assignments).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Build the response list
	var responses []map[string]interface{}

	// First, determine the latest active assignment for each patient-form combination
	// This will help us identify which assignment is the latest active one
	latestActiveAssignments := make(map[string]uint) // key: "patientID_formID", value: assignmentID

	for _, a := range assignments {
		// Skip archived assignments when determining latest active
		if a.Status == models.EConsentStatusArchived {
			continue
		}

		key := fmt.Sprintf("%d_%d", a.PatientID, a.EConsentFormID)
		if existingAssignmentID, exists := latestActiveAssignments[key]; exists {
			// Compare version numbers to determine which is latest
			var existingVersion, currentVersion int
			config.DB.Model(&models.EConsentFormVersion{}).
				Where("id = ?", a.EConsentFormVersionID).
				Select("version_number").Scan(&currentVersion)

			// Get the existing assignment's version
			var existingAssignment models.EConsentAssignment
			if err := config.DB.Where("id = ?", existingAssignmentID).First(&existingAssignment).Error; err == nil {
				config.DB.Model(&models.EConsentFormVersion{}).
					Where("id = ?", existingAssignment.EConsentFormVersionID).
					Select("version_number").Scan(&existingVersion)
			}

			// If current version is higher, update the latest
			if currentVersion > existingVersion {
				latestActiveAssignments[key] = a.ID
			}
		} else {
			latestActiveAssignments[key] = a.ID
		}
	}

	for _, a := range assignments {
		// Encrypt patient's name and email before building response
		encryptedFirstName, _ := utils.Encrypt(a.User.FirstName)
		encryptedLastName, _ := utils.Encrypt(a.User.LastName)
		encryptedEmail, _ := utils.Encrypt(a.User.Email)

		patients := []map[string]interface{}{
			{
				"name":  strings.TrimSpace(encryptedFirstName + " " + encryptedLastName),
				"id":    a.User.ID,
				"email": encryptedEmail,
			},
		}

		// Build coordinators array for this assignment
		coordinators := []map[string]interface{}{}

		// Unmarshal CoordinatorIDs if needed
		var coordinatorIDs []uint
		switch v := any(a.EConsentForm.CoordinatorIDs).(type) {
		case string:
			if err := json.Unmarshal([]byte(v), &coordinatorIDs); err != nil {
				fmt.Printf("Failed to unmarshal coordinator_ids: %v\n", err)
			}
		case []uint:
			coordinatorIDs = v
		case models.PatientAssignedArray:
			coordinatorIDs = []uint(v)
		default:
			fmt.Printf("Unknown type for CoordinatorIDs: %T\n", v)
		}

		if len(coordinatorIDs) > 0 {
			var coordinatorUsers []models.User
			if err := config.DB.Where("id IN (?)", coordinatorIDs).Find(&coordinatorUsers).Error; err == nil {
				for _, u := range coordinatorUsers {
					encryptedFirstName, _ := utils.Encrypt(u.FirstName)
					encryptedLastName, _ := utils.Encrypt(u.LastName)
					encryptedEmail, _ := utils.Encrypt(u.Email)
					coordinators = append(coordinators, map[string]interface{}{
						"name":  strings.TrimSpace(encryptedFirstName + " " + encryptedLastName),
						"id":    u.ID,
						"email": encryptedEmail,
					})
				}
			} else {
				fmt.Printf("Coordinator user query error: %v\n", err)
			}
		}

		var patientEConsentID *uint
		// Always try to find a PatientEConsent record for the assignment.
		var pe models.PatientEConsent
		result := config.DB.Where("patient_id = ? AND e_consent_form_id = ?", a.PatientID, a.EConsentFormID).Order("created_at desc").First(&pe)
		if result.Error == nil {
			patientEConsentID = &pe.ID
		}

		// Prepare withdrawal info if status is withdrawn or reviewed
		var withdrawalInfo map[string]interface{}
		if a.Status == models.EConsentStatusWithdrawn || a.Status == models.EConsentStatusReviewed {
			withdrawalInfo = map[string]interface{}{
				"withdrawn_reason":      a.WithdrawalReason,
				"withdrawn_reviewed_by": a.WithdrawnReviewedBy,
				"withdrawn_reviewed_at": a.WithdrawnReviewedAt,
			}
		}

		var versionNumber int
		if a.EConsentFormVersionID != 0 {
			config.DB.Model(&models.EConsentFormVersion{}).
				Where("id = ?", a.EConsentFormVersionID).
				Select("version_number").Scan(&versionNumber)
		}

		// Determine if this is the latest active assignment for this patient-form combination
		key := fmt.Sprintf("%d_%d", a.PatientID, a.EConsentFormID)
		isLatestActive := latestActiveAssignments[key] == a.ID

		// Get expiry time for this assignment's consent form (if available)
		var expiryTime *time.Time
		if !a.EConsentForm.ExpiryTime.IsZero() {
			expiryTime = &a.EConsentForm.ExpiryTime
		}

		resp := map[string]interface{}{
			"assignment_id":       a.ID,
			"patient_id":          a.PatientID,
			"patient_econsent_id": patientEConsentID,
			"econsent_form_id":    a.EConsentFormID,
			"econsent_title":      a.EConsentForm.Title,
			"assigned_at":         a.AssignedAt.Format(time.RFC3339),
			"status":              string(a.Status),
			"patients":            patients,
			"coordinators":        coordinators,
			"version":             versionNumber,
			"is_latest_active":    isLatestActive,
			"expiry_time":         nil,
		}
		if expiryTime != nil {
			resp["expiry_time"] = expiryTime.Format(time.RFC3339)
		}
		if withdrawalInfo != nil {
			resp["withdrawal"] = withdrawalInfo
		}
		responses = append(responses, resp)
	}

	// Prepare the paginated response structure
	response := PatientConsentAssignmentListResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      nil, // will set below
	}
	// Set Records as []map[string]interface{} instead of []PatientConsentAssignmentResponse
	// so we marshal manually
	utils.JSONResponse(c, http.StatusOK, "Patient consent assignments fetched successfully", map[string]interface{}{
		"page":          response.Page,
		"per_page":      response.PerPage,
		"sort":          response.Sort,
		"sort_column":   response.SortColumn,
		"search_text":   response.SearchText,
		"total_records": response.TotalRecords,
		"total_pages":   response.TotalPages,
		"records":       responses,
	})
}

func GeneratePatientConsentPDFHandler(c *gin.Context) {
	idStr := c.Param("patient_econsent_id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid patient_econsent_id", nil)
		return
	}

	// Fetch PatientEConsent with related data
	var pe models.PatientEConsent
	err = config.DB.
		Preload("Patient").
		Preload("Nurse").
		Preload("EConsentForm.Research").
		Preload("EConsentForm.Declarations.Declaration").
		Preload("EConsentForm.DeclarationForms.Declarations").
		Where("id = ? AND is_deleted = ?", id, false).
		First(&pe).Error
	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Patient e-consent not found", nil)
		return
	}

	// Fetch declaration responses for this patient e-consent
	var declResponses []models.DeclarationResponse
	dbResult := config.DB.Preload("Declaration").Where("patient_e_consent_id = ?", pe.ID).Find(&declResponses)
	if dbResult.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch declaration responses", nil)
		return
	}

	// Map to PDF DTO
	pdfReq := dto.ConsentFormPDFRequest{
		NHI:                 "",
		FirstName:           "",
		LastName:            "",
		DOB:                 "",
		Gender:              "",
		ProjectName:         pe.EConsentForm.Research.Title,
		QuestionnaireName:   pe.EConsentForm.Title,
		PatientSignature:    pe.PatientSignURL,
		PatientSignDate:     "",
		ResearcherName:      pe.Nurse.FirstName + " " + pe.Nurse.LastName,
		ResearcherSignature: pe.NurseSignURL,
		ResearcherSignDate:  "",
		TrackDate:           pe.CreatedAt.Format("2006-01-02"),
		CompletedAt:         pe.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if pe.NHI != "" {
		decryptedNHI, err := utils.Decrypt(pe.NHI)
		if err == nil {
			pdfReq.NHI = decryptedNHI
		}
	}
	if pe.FirstName != "" {
		decryptedFirstName, err := utils.Decrypt(pe.FirstName)
		if err == nil {
			pdfReq.FirstName = decryptedFirstName
		}
	}
	if pe.LastName != "" {
		decryptedLastName, err := utils.Decrypt(pe.LastName)
		if err == nil {
			pdfReq.LastName = decryptedLastName
		}
	}
	if pe.Gender != "" {
		decryptedGender, err := utils.Decrypt(pe.Gender)
		if err == nil {
			pdfReq.Gender = decryptedGender
		}
	}

	if pe.DOB != "" {
		decryptedDOB, err := utils.Decrypt(pe.DOB)
		if err == nil {
			// If you want to ensure it's in "YYYY-MM-DD" format:
			parsedDOB, err := time.Parse("2006-01-02", decryptedDOB)
			if err == nil {
				pdfReq.DOB = parsedDOB.Format("2006-01-02")
			} else {
				pdfReq.DOB = decryptedDOB // fallback: use as-is
			}
		} else {
			pdfReq.DOB = "" // or handle error as needed
		}
	}
	if pe.PatientSignedAt != nil {
		pdfReq.PatientSignDate = pe.PatientSignedAt.Format("2006-01-02")
	}
	if pe.NurseSignedAt != nil {
		pdfReq.ResearcherSignDate = pe.NurseSignedAt.Format("2006-01-02")
	}

	log.Printf("DeclarationForms for PDF: %+v")
	// Map declaration responses to PDF fields dynamically (grouped by form)
	declarationForms := []dto.DeclarationFormPDFInfo{}
	for _, form := range pe.EConsentForm.DeclarationForms {
		formInfo := dto.DeclarationFormPDFInfo{
			FormID:    form.ID,
			FormLabel: form.Name,
		}
		for _, decl := range form.Declarations {
			var respText string
			for _, dr := range declResponses {
				if dr.DeclarationID == decl.ID && dr.Response != "" {
					decryptedResp, err := utils.Decrypt(dr.Response)
					if err == nil {
						// Check declaration type: if boolean, map to YES/NO, else use text as-is
						if decl.ResponseType == "yes_no" {
							if decryptedResp == "true" {
								respText = "YES"
							} else if decryptedResp == "false" {
								respText = "NO"
							} else {
								respText = "UNKNOWN"
							}
						} else {
							respText = decryptedResp
						}
					} else {
						respText = "UNKNOWN"
					}
					break
				}
			}
			formInfo.Declarations = append(formInfo.Declarations, dto.DeclarationPDFInfo{
				Statement: decl.Statement,
				Response:  respText,
			})
		}
		declarationForms = append(declarationForms, formInfo)
	}
	log.Printf("DeclarationForms for PDF: %+v", declarationForms)
	pdfReq.DeclarationForms = declarationForms

	pdfDir := "public/econsents/pdf/"
	_ = os.MkdirAll(pdfDir, 0755)
	pdfFileName := fmt.Sprintf("consent_%d_v%d.pdf", pe.ID, pe.EConsentFormVersionID)
	pdfFilePath := pdfDir + pdfFileName
	// Always regenerate the PDF
	pdfBytes, err := utils.GenerateConsentPDF(pdfReq)
	if err == nil {
		_ = os.WriteFile(pdfFilePath, pdfBytes, 0644)
	}
	// if pdfBytes != nil {
	// 	var patient models.User
	// 	if err := config.DB.Where("id = ?", pe.PatientID).First(&patient).Error; err == nil {
	// 		emailBody := emails.PatientConsentApprovedEmail(patient.FirstName, patient.LastName)
	// 		subject := "Your E-Consent Form Has Been Approved"
	// 		recipients := []string{patient.Email}
	// 		_ = config.SendEmailWithAttachment(recipients, subject, emailBody, pdfFileName, pdfBytes)
	// 	}
	// }

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=consent_form.pdf")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// RequestResignPatientEConsent allows admin/nurse to request re-signing of a patient e-consent form
func RequestResignPatientEConsent(c *gin.Context) {
	var err error
	// Only admin or nurse can perform this action
	user, ok := middlewares.GetUserFromContext(c)
	//if !ok || (user.RoleID != 1 && user.RoleID != 2) {
	if !ok || (user.RoleID != 5) {
		utils.JSONResponse(c, http.StatusForbidden, "Only admin or super admin can request re-signing", nil)
		return
	}

	// Get patient_econsent_id from request (can be in JSON or query)
	var req struct {
		PatientEConsentID uint `json:"patient_econsent_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Missing or invalid patient_econsent_id", nil)
		return
	}

	// Find the PatientEConsent record
	var pe models.PatientEConsent
	if err := config.DB.Where("id = ? AND is_deleted = ?", req.PatientEConsentID, false).First(&pe).Error; err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Patient e-consent not found", nil)
		return
	}

	// Fetch the EConsentForm to get the expiry time
	var eConsentForm models.EConsentForm
	if err := config.DB.Where("id = ?", pe.EConsentFormID).First(&eConsentForm).Error; err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "E-consent form not found", nil)
		return
	}

	expired, msg := services.CheckEConsentExpiry(config.DB, pe.EConsentFormID, &pe.ID, nil)
	if expired {
		utils.JSONResponse(c, http.StatusForbidden, msg, nil)
		return
	}

	// Only allow if current status is 'submitted'
	if pe.Status != models.EConsentStatusSubmitted {
		utils.JSONResponse(c, http.StatusBadRequest, "Can only request re-signing if status is 'submitted'", nil)
		return
	}

	// Start transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Update PatientEConsent in transaction
	pe.Status = models.EConsentStatusInProgress
	if err := tx.Save(&pe).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update status", nil)
		return
	}

	// Update EConsentAssignment in transaction
	var assignment models.EConsentAssignment
	if err := tx.Where("patient_id = ? AND e_consent_form_id = ? AND e_consent_form_version_id = ? AND is_deleted = ?", pe.PatientID, pe.EConsentFormID, pe.EConsentFormVersionID, false).First(&assignment).Error; err == nil {
		assignment.Status = models.EConsentStatusInProgress // or use a custom status if needed
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

	// Send notification to the patient about counter sign request
	err = services.CreateNotification(
		config.DB,
		pe.PatientID,
		"Counter Sign Requested",
		"You are requested to re submit your signature for your e-consent form. Please log in to your portal to complete the process.",
		"E-Consent",
		user.FirstName+" "+user.LastName,
		"econsents",
		pe.ID,
		map[string]interface{}{
			"econsent_id": pe.ID,
			"status":      "counter_sign_requested",
		},
	)
	if err != nil {
		log.Printf("Failed to notify patient about counter sign request: %v", err)
	}

	// Send re-signing request email to patient
	var patient models.User
	if err := config.DB.Where("id = ?", pe.PatientID).First(&patient).Error; err == nil {
		// Fetch the e-consent form title
		var eConsentForm models.EConsentForm
		if err := config.DB.Where("id = ?", pe.EConsentFormID).First(&eConsentForm).Error; err == nil {
			patientName := strings.TrimSpace(patient.FirstName + " " + patient.LastName)
			eConsentTitle := eConsentForm.Title
			portalLink := config.AppConfig.AppUrl + "/login"
			emailBody := emails.PatientResignRequestEmail(patientName, eConsentTitle, portalLink)
			subject := "Action Required: Please Re-sign Your E-Consent Form"
			recipients := []string{patient.Email}
			if err := config.SendEmail(recipients, subject, emailBody); err != nil {
				fmt.Printf("Failed to send re-signing email to patient: %v\n", err)
			}
		}
	}

	utils.JSONResponse(c, http.StatusOK, "Re-signing requested. Patient must re-sign the e-consent form.", nil)
}

// Admin/Super Admin sign (approve) a patient e-consent form that is first signed by patient
func AdminSignPatientEConsent(c *gin.Context) {
	// Only admin or super admin can perform this action
	user, ok := middlewares.GetUserFromContext(c)
	//if !ok || (user.RoleID != 1 && user.RoleID != 2) {
	if !ok || (user.RoleID != 5) {
		utils.JSONResponse(c, http.StatusForbidden, "Only admin or super admin can sign patient e-consent forms", nil)
		return
	}

	// Get patient_econsent_id from URL param
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

	// Only allow if current status is 'submitted' (i.e., signed by patient)
	if pe.Status == models.EConsentStatusApproved {
		utils.JSONResponse(c, http.StatusOK, "This consent has already been approved.", nil)
		return
	}

	if pe.Status != models.EConsentStatusSubmitted {
		utils.JSONResponse(c, http.StatusBadRequest, "Can only sign if already signed by patient", nil)
		return
	}

	// Handle file upload (signature image) or reuse nurse's default
	var signatureURL, signatureHash string
	if file, err := c.FormFile("signature"); err == nil && file != nil {
		// Compute hash of uploaded file
		hash, fileBytes, err := utils.ComputeFileHashFromFormFile(file)
		if err != nil {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to compute signature hash", nil)
			return
		}
		signatureDir := "public/signatures/nurse/"
		_ = os.MkdirAll(signatureDir, 0755)
		filename := fmt.Sprintf("nurse_%d_%s%s", user.ID, hash, utils.GetFileExtension(file.Filename))
		filePath := signatureDir + filename
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := os.WriteFile(filePath, fileBytes, 0644); err != nil {
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save signature image", nil)
				return
			}
		}
		signatureURL = "/signatures/nurse/" + filename // public URL
		signatureHash = hash
		// Update nurse's default signature URL and hash
		user.DefaultNurseSignatureURL = signatureURL
		user.DefaultNurseSignatureHash = hash
		_ = config.DB.Save(&user)
	} else if user.DefaultNurseSignatureURL != "" && user.DefaultNurseSignatureHash != "" {
		signatureURL = user.DefaultNurseSignatureURL
		signatureHash = user.DefaultNurseSignatureHash
	} else {
		utils.JSONResponse(c, http.StatusBadRequest, "A nurse signature is required.", nil)
		return
	}

	// Update status to 'approved', set nurse info and signature URL/hash
	pe.Status = models.EConsentStatusApproved
	pe.NurseID = user.ID
	pe.NurseSignURL = signatureURL
	pe.NurseSignHash = signatureHash
	now := time.Now()
	pe.NurseSignedAt = &now

	// Start transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Update PatientEConsent in transaction
	if err := tx.Save(&pe).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to approve/sign e-consent form", nil)
		return
	}

	// Update EConsentAssignment in transaction
	var assignment models.EConsentAssignment
	if err := tx.Where("patient_id = ? AND e_consent_form_id = ? AND e_consent_form_version_id = ? AND is_deleted = ?", pe.PatientID, pe.EConsentFormID, pe.EConsentFormVersionID, false).First(&assignment).Error; err == nil {
		assignment.Status = models.EConsentStatusApproved
		now := time.Now()
		assignment.ApprovedAt = &now
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

	// Send approval email with PDF to patient (after commit)
	var patient models.User
	if err := config.DB.Where("id = ?", pe.PatientID).First(&patient).Error; err == nil {
		// Fetch declaration responses for this patient e-consent
		var declResponses []models.DeclarationResponse
		dbResult := config.DB.Preload("Declaration").Where("patient_e_consent_id = ?", pe.ID).Find(&declResponses)
		if dbResult.Error == nil {
			// Map to PDF DTO
			pdfReq := dto.ConsentFormPDFRequest{
				NHI:                 "",
				FirstName:           "",
				LastName:            "",
				DOB:                 "",
				Gender:              "",
				ProjectName:         pe.EConsentForm.Research.Title,
				QuestionnaireName:   pe.EConsentForm.Title,
				PatientSignature:    pe.PatientSignURL,
				PatientSignDate:     "",
				ResearcherName:      user.FirstName + " " + user.LastName,
				ResearcherSignature: pe.NurseSignURL,
				ResearcherSignDate:  "",
			}
			if pe.NHI != "" {
				decryptedNHI, err := utils.Decrypt(pe.NHI)
				if err == nil {
					pdfReq.NHI = decryptedNHI
				}
			}
			if pe.FirstName != "" {
				decryptedFirstName, err := utils.Decrypt(pe.FirstName)
				if err == nil {
					pdfReq.FirstName = decryptedFirstName
				}
			}
			if pe.LastName != "" {
				decryptedLastName, err := utils.Decrypt(pe.LastName)
				if err == nil {
					pdfReq.LastName = decryptedLastName
				}
			}
			if pe.Gender != "" {
				decryptedGender, err := utils.Decrypt(pe.Gender)
				if err == nil {
					pdfReq.Gender = decryptedGender
				}
			}
			if pe.DOB != "" {
				decryptedDOB, err := utils.Decrypt(pe.DOB)
				if err == nil {
					// If you want to ensure it's in "YYYY-MM-DD" format:
					parsedDOB, err := time.Parse("2006-01-02", decryptedDOB)
					if err == nil {
						pdfReq.DOB = parsedDOB.Format("2006-01-02")
					} else {
						pdfReq.DOB = decryptedDOB // fallback: use as-is
					}
				} else {
					pdfReq.DOB = "" // or handle error as needed
				}
			}
			if pe.PatientSignedAt != nil {
				pdfReq.PatientSignDate = pe.PatientSignedAt.Format("2006-01-02")
			}
			if pe.NurseSignedAt != nil {
				pdfReq.ResearcherSignDate = pe.NurseSignedAt.Format("2006-01-02")
			}
			// Map declaration responses to PDF fields dynamically
			declarations := []dto.DeclarationPDFInfo{}
			for _, consentDecl := range pe.EConsentForm.Declarations {
				// Find the response for this declaration
				var respText string
				for _, dr := range declResponses {
					if dr.DeclarationID == consentDecl.DeclarationID && dr.Response != "" {
						decryptedResp, err := utils.Decrypt(dr.Response)
						if err == nil && decryptedResp == "true" {
							respText = "YES"
						} else if err == nil && decryptedResp == "false" {
							respText = "NO"
						} else {
							respText = "UNKNOWN"
						}
						break
					}
				}
				declarations = append(declarations, dto.DeclarationPDFInfo{
					Statement: consentDecl.Declaration.Statement,
					Response:  respText,
				})
			}
			// pdfReq.Declarations = declarations
			pdfDir := "public/econsents/pdf/"
			_ = os.MkdirAll(pdfDir, 0755)
			pdfFileName := fmt.Sprintf("consent_%d_v%d.pdf", pe.ID, pe.EConsentFormVersionID)
			pdfFilePath := pdfDir + pdfFileName
			// Always regenerate the PDF
			pdfBytes, err := utils.GenerateConsentPDF(pdfReq)
			if err == nil {
				_ = os.WriteFile(pdfFilePath, pdfBytes, 0644)
			}
			if pdfBytes != nil {
				emailBody := emails.PatientConsentApprovedEmail(patient.FirstName, patient.LastName)
				subject := "Your E-Consent Form Has Been Approved"
				recipients := []string{patient.Email}
				_ = config.SendEmailWithAttachment(recipients, subject, emailBody, pdfFileName, pdfBytes)
			}
		}
	}

	// Send notification to the patient about consent approval
	err = services.CreateNotification(
		config.DB,
		pe.PatientID,
		"Consent Approved",
		"Your e-consent form has been approved by the nurse. Please check your email for the signed PDF.",
		"E-Consent",
		user.FirstName+" "+user.LastName,
		"econsents",
		pe.ID,
		map[string]interface{}{
			"econsent_id": pe.ID,
			"status":      "approved",
		},
	)
	if err != nil {
		log.Printf("Failed to notify patient about consent approval: %v", err)
	}

	// Notify all other coordinators about the consent approval
	var eConsentForm models.EConsentForm
	if err := config.DB.Where("id = ?", pe.EConsentFormID).First(&eConsentForm).Error; err == nil {
		for _, coordinatorID := range eConsentForm.CoordinatorIDs {
			// Skip the current coordinator who just approved
			if coordinatorID == user.ID {
				continue
			}

			// Send notification to other coordinators
			err = services.CreateNotification(
				config.DB,
				coordinatorID,
				"Patient Consent Approved by Coordinator",
				fmt.Sprintf("Patient %s %s's consent for e-consent form '%s' has been approved by %s %s.",
					patient.FirstName, patient.LastName, eConsentForm.Title, user.FirstName, user.LastName),
				"E-Consent-Approval",
				user.FirstName+" "+user.LastName,
				"econsents",
				pe.ID,
				map[string]interface{}{
					"econsent_id": pe.ID,
					"status":      "approved",
					"approved_by": user.ID,
					"patient_id":  pe.PatientID,
				},
			)
			if err != nil {
				log.Printf("Failed to notify coordinator %d about consent approval: %v", coordinatorID, err)
			}
		}
	}

	utils.JSONResponse(c, http.StatusOK, "E-consent form approved by admin successfully", map[string]interface{}{
		"nurse_sign_url": signatureURL,
	})
}

// NurseReviewWithdrawnConsent allows a nurse to mark a withdrawn consent as reviewed
func NurseReviewWithdrawnConsent(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok || (user.RoleID != 1 && user.RoleID != 2) {
		utils.JSONResponse(c, http.StatusForbidden, "Only nurse can manage the withdrawls", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid patient_econsent_id", nil)
		return
	}

	var pe models.PatientEConsent
	if err := config.DB.Where("id = ? AND is_deleted = ?", id, false).First(&pe).Error; err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Patient e-consent not found", nil)
		return
	}

	if pe.Status != models.EConsentStatusWithdrawn {
		utils.JSONResponse(c, http.StatusBadRequest, "Consent is not in withdrawn status", nil)
		return
	}

	tx := config.DB.Begin()
	if tx.Error != nil {

		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	now := time.Now()
	pe.Status = models.EConsentStatusReviewed
	pe.WithdrawnReviewedBy = user.ID
	pe.WithdrawnReviewedAt = &now
	if err := tx.Save(&pe).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to update consent: %v", err)
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update consent", nil)
		return
	}

	// Update ALL withdrawn consents for this patient and e-consent form (including previous versions)
	var allWithdrawnConsents []models.PatientEConsent
	if err := tx.Where("patient_id = ? AND e_consent_form_id = ? AND status = ? AND is_deleted = ?",
		pe.PatientID, pe.EConsentFormID, models.EConsentStatusWithdrawn, false).Find(&allWithdrawnConsents).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch withdrawn consents", nil)
		return
	}

	// Update all withdrawn consents to reviewed status
	for _, consent := range allWithdrawnConsents {
		consent.Status = models.EConsentStatusReviewed
		consent.WithdrawnReviewedBy = user.ID
		consent.WithdrawnReviewedAt = &now
		if err := tx.Save(&consent).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update consent status", nil)
			return
		}
	}

	// Update ALL withdrawn assignments for this patient and e-consent form (including previous versions)
	var allWithdrawnAssignments []models.EConsentAssignment
	if err := tx.Where("patient_id = ? AND e_consent_form_id = ? AND status = ? AND is_deleted = ?",
		pe.PatientID, pe.EConsentFormID, models.EConsentStatusWithdrawn, false).Find(&allWithdrawnAssignments).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch withdrawn assignments", nil)
		return
	}

	// Update all withdrawn assignments to reviewed status
	for _, assignment := range allWithdrawnAssignments {
		assignment.Status = models.EConsentStatusReviewed
		assignment.WithdrawnReviewedBy = user.ID
		assignment.WithdrawnReviewedAt = &now
		if err := tx.Save(&assignment).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update assignment status", nil)
			return
		}
	}

	// Send notification to the patient about consent review
	err = services.CreateNotification(
		config.DB,
		pe.PatientID,
		"Consent Withdrawn",
		"Your withdrawn request for e-consent form has been approved by the nurse.",
		"E-Consent",
		user.FirstName+" "+user.LastName,
		"econsents",
		pe.ID,
		map[string]interface{}{
			"econsent_id": pe.ID,
			"status":      "reviewed",
		},
	)
	if err != nil {
		log.Printf("Failed to notify patient about consent review: %v", err)
	}

	// Send email to patient about consent review
	var patient models.User
	if err := config.DB.Where("id = ?", pe.PatientID).First(&patient).Error; err == nil {
		emailBody := emails.PatientConsentWithdrawnReviewedEmail(patient.FirstName + " " + patient.LastName)
		subject := "Your Withdrawn E-Consent Form Has Been Reviewed"
		recipients := []string{patient.Email}
		if err := config.SendEmail(recipients, subject, emailBody); err != nil {
			log.Printf("Failed to send consent review email to patient: %v", err)
		}
	}

	// Notify all other coordinators about the consent review
	var eConsentForm models.EConsentForm
	if err := config.DB.Where("id = ?", pe.EConsentFormID).First(&eConsentForm).Error; err == nil {
		for _, coordinatorID := range eConsentForm.CoordinatorIDs {
			// Skip the current coordinator who just reviewed
			if coordinatorID == user.ID {
				continue
			}

			// Send notification to other coordinators
			err = services.CreateNotification(
				config.DB,
				coordinatorID,
				"Patient Consent Withdrawal Reviewed by Coordinator",
				fmt.Sprintf("Patient %s %s's withdrawn consent for e-consent form '%s' has been reviewed by %s %s.",
					patient.FirstName, patient.LastName, eConsentForm.Title, user.FirstName, user.LastName),
				"E-Consent-Review",
				user.FirstName+" "+user.LastName,
				"econsents",
				pe.ID,
				map[string]interface{}{
					"econsent_id": pe.ID,
					"status":      "reviewed",
					"reviewed_by": user.ID,
					"patient_id":  pe.PatientID,
				},
			)
			if err != nil {
				log.Printf("Failed to notify coordinator %d about consent review: %v", coordinatorID, err)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, "Consent marked as reviewed", nil)
}
