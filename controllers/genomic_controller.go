package controllers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"
)

type PatientResult struct {
	PatientID            uint   `json:"patient_id"`
	EConsentFormID       uint   `json:"econsent_form_id"`
	EConsentFormVersionID uint   `json:"econsent_form_version_id"`
	VersionNumber        int    `json:"version_number"`
	PatientName          string `json:"patient_name"`
	ResearchId           uint   `json:"research_id"`
	ResearchTitle        string `json:"research_title"`
	Status               bool   `json:"status"`
	ResultsPublished     bool   `json:"results_published"`
	EconsentTitle        string `json:"econsent_title"`
	SubmittedAt          string `json:"submitted_at"`
	ConsentStatus        string `json:"consent_status"`        // "approved", "withdrawn", "reviewed"
	CanPublishResults    bool   `json:"can_publish_results"`   // Flag to control UI publishing ability
}

type GenomicResultListResponse[T any] struct {
	Page         int    `json:"page"`
	PerPage      int    `json:"per_page"`
	Sort         string `json:"sort"`
	SortColumn   string `json:"sort_column"`
	SearchText   string `json:"search_text"`
	TotalRecords int64  `json:"total_records"`
	TotalPages   int    `json:"total_pages"`
	Records      []T    `json:"records"`
}

type GenomicResultResponse struct {
	ID                    uint   `json:"id"`
	PatientID             uint   `json:"patient_id"`
	EConsentFormID        uint   `json:"econsent_form_id"`
	EConsentFormVersionID *uint  `json:"econsent_form_version_id,omitempty"`
	VersionNumber         *int   `json:"version_number,omitempty"`
	PatientName           string `json:"patient_name"`
	ResearchTitle         string `json:"research_title"`
	EConsentTitle         string `json:"econsent_title"`
	FileURL               string `json:"file_url"`
	IsPublished           bool   `json:"is_published"`
	CreatedAt             string `json:"created_at"`
	ConsentStatus         string `json:"consent_status"`  // "approved", "withdrawn", "reviewed"
}

// ListConsentedPatients lists patients who have consented to a study
func ListConsentedPatients(c *gin.Context) {
	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text"}

	// Parse query parameters with default values
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid query parameters", nil)
		return
	}

	// Default and validation for 'page'
	page := 1
	if val := query.Get("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid page parameter", nil)
			return
		}
	}

	// Default and validation for 'per_page'
	perPage := 10
	if val := query.Get("per_page"); val != "" {
		if pp, err := strconv.Atoi(val); err == nil && pp > 0 {
			perPage = pp
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid per_page parameter", nil)
			return
		}
	}

	// Default and validation for 'sort'
	sortOrder := "desc"
	if val := strings.ToLower(query.Get("sort")); val == "asc" || val == "desc" {
		sortOrder = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid sort parameter", nil)
		return
	}

	// Default and validation for 'sort_column'
	sortColumn := "submitted_at"
	validSortColumns := []string{"submitted_at", "research_title", "econsent_title", "patient_name"}
	if val := strings.ToLower(query.Get("sort_column")); val != "" {
		if utils.IsInList(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid sort_column parameter", nil)
			return
		}
	}

	// Optional 'search_text'
	searchText := strings.TrimSpace(query.Get("search_text"))

	// Hybrid approach: Original simple logic + Signature filter
	// 1. Original logic: Show non-assigned statuses (simple & clean)
	// 2. Signature filter: Ensure only patients who participated (have nurse signatures) OR have published genomic results
	// This prevents non-participating patients (like P2) from showing up after versioning
	db := config.DB.Model(&models.EConsentAssignment{}).
		Preload("User").
		Preload("EConsentForm.Research").
		Preload("EConsentFormVersion").
		Where("e_consent_assignments.is_deleted = ? AND e_consent_assignments.status != ? AND (EXISTS (SELECT 1 FROM patient_e_consents WHERE patient_e_consents.patient_id = e_consent_assignments.patient_id AND patient_e_consents.e_consent_form_id = e_consent_assignments.e_consent_form_id AND patient_e_consents.nurse_sign_url IS NOT NULL AND patient_e_consents.is_deleted = false) OR EXISTS (SELECT 1 FROM genomics WHERE genomics.patient_id = e_consent_assignments.patient_id AND genomics.e_consent_form_id = e_consent_assignments.e_consent_form_id AND genomics.is_published = true AND genomics.is_deleted = false))", false, models.EConsentStatusAssigned)

	if searchText != "" {
		db = db.Joins("JOIN e_consent_forms ON e_consent_forms.id = e_consent_assignments.e_consent_form_id").
			Joins("JOIN researches ON researches.id = e_consent_forms.research_id").
			Where("researches.title ILIKE ? OR e_consent_forms.title ILIKE ?", "%"+searchText+"%", "%"+searchText+"%")
	}

	// 3. Count total records - include all versions to preserve genomic results visibility
	var totalRecords int64
	db.Count(&totalRecords)

	// 4. Map sort column to actual database columns
	var sortExpr string
	switch sortColumn {
	case "research_title":
		if searchText != "" {
			sortExpr = "researches.title " + sortOrder + ", COALESCE(e_consent_assignments.submitted_at, e_consent_assignments.assigned_at) " + sortOrder + ", e_consent_assignments.id " + sortOrder
		} else {
			// If no search, we need to join for sorting
			db = db.Joins("JOIN e_consent_forms ON e_consent_forms.id = e_consent_assignments.e_consent_form_id").
				Joins("JOIN researches ON researches.id = e_consent_forms.research_id")
			sortExpr = "researches.title " + sortOrder + ", COALESCE(e_consent_assignments.submitted_at, e_consent_assignments.assigned_at) " + sortOrder + ", e_consent_assignments.id " + sortOrder
		}
	case "econsent_title":
		if searchText != "" {
			sortExpr = "e_consent_forms.title " + sortOrder + ", COALESCE(e_consent_assignments.submitted_at, e_consent_assignments.assigned_at) " + sortOrder + ", e_consent_assignments.id " + sortOrder
		} else {
			// If no search, we need to join for sorting
			db = db.Joins("JOIN e_consent_forms ON e_consent_forms.id = e_consent_assignments.e_consent_form_id")
			sortExpr = "e_consent_forms.title " + sortOrder + ", COALESCE(e_consent_assignments.submitted_at, e_consent_assignments.assigned_at) " + sortOrder + ", e_consent_assignments.id " + sortOrder
		}
	case "patient_name":
		sortExpr = "users.first_name " + sortOrder + ", users.last_name " + sortOrder + ", COALESCE(e_consent_assignments.submitted_at, e_consent_assignments.assigned_at) " + sortOrder + ", e_consent_assignments.id " + sortOrder
	case "submitted_at":
		fallthrough
	default:
		// Ensure deterministic sorting by adding secondary sort keys
		// Use COALESCE to handle NULL submitted_at by falling back to assigned_at
		sortExpr = "COALESCE(e_consent_assignments.submitted_at, e_consent_assignments.assigned_at) " + sortOrder + ", e_consent_assignments.id " + sortOrder
	}

	// 5. Pagination and sorting
	offset := (page - 1) * perPage
	db = db.Order(sortExpr).Limit(perPage).Offset(offset)

	// 6. Fetch records - include all versions to preserve genomic results visibility
	var assignments []models.EConsentAssignment
	db.Find(&assignments)

	// 7. Build response records
	var results []PatientResult
	for _, assignment := range assignments {
		// Handle nil User field safely
		var name string
		if assignment.User.ID != 0 {
			name = strings.TrimSpace(assignment.User.FirstName + " " + assignment.User.LastName)
		} else {
			name = "Unknown Patient"
		}
		
		// Check if published genomic result exists for this patient and e-consent form
		// Include results from all versions, not just the current one
		var resultCount int64
		config.DB.Model(&models.Genomics{}).
			Where("patient_id = ? AND e_consent_form_id = ? AND is_published = ? AND is_deleted = ?", assignment.PatientID, assignment.EConsentFormID, true, false).
			Count(&resultCount)
		resultStatus := resultCount > 0
		// If no results found for current version, check if there are results for any version of this form
		if !resultStatus {
			var anyVersionResultCount int64
			config.DB.Model(&models.Genomics{}).
				Where("patient_id = ? AND e_consent_form_id = ? AND is_published = ? AND is_deleted = ?", assignment.PatientID, assignment.EConsentFormID, true, false).
				Count(&anyVersionResultCount)
			resultStatus = anyVersionResultCount > 0
		}
		
		// Get the version number from preloaded data
		versionNumber := 0
		if assignment.EConsentFormVersion.ID != 0 {
			versionNumber = assignment.EConsentFormVersion.VersionNumber
		}
		
		encryptedName, encErr := utils.Encrypt(name)
		if encErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt patient name"})
			return
		}
		// Determine if results can be published based on consent status
		canPublishResults := assignment.Status == models.EConsentStatusApproved
		
		// Determine if the consent is active based on status
		// Only approved consents are considered active
		isActive := assignment.Status == models.EConsentStatusApproved
		
		// Handle nil SubmittedAt field safely
		var submittedAtStr string
		if assignment.SubmittedAt != nil {
			submittedAtStr = assignment.SubmittedAt.Format(time.RFC3339)
		} else {
			// Use AssignedAt as fallback if SubmittedAt is nil
			submittedAtStr = assignment.AssignedAt.Format(time.RFC3339)
			fmt.Printf("DEBUG: Using AssignedAt fallback for PatientID=%d, FormID=%d (SubmittedAt is nil)\n", 
				assignment.PatientID, assignment.EConsentFormID)
		}

		// Handle nil relationship fields safely
		var researchID uint
		var researchTitle string
		var econsentTitle string
		
		if assignment.EConsentForm.ID != 0 {
			researchID = assignment.EConsentForm.ResearchID
			econsentTitle = assignment.EConsentForm.Title
			
			if assignment.EConsentForm.Research.ID != 0 {
				researchTitle = assignment.EConsentForm.Research.Title
			}
		}

		results = append(results, PatientResult{
			PatientID:            assignment.PatientID,
			EConsentFormID:       assignment.EConsentFormID,
			EConsentFormVersionID: assignment.EConsentFormVersionID,
			VersionNumber:        versionNumber,
			PatientName:          encryptedName,
			ResearchId:           researchID,
			ResearchTitle:        researchTitle,
			Status:               isActive,
			EconsentTitle:        econsentTitle,
			ResultsPublished:     resultStatus,
			SubmittedAt:          submittedAtStr,
			ConsentStatus:        string(assignment.Status),
			CanPublishResults:    canPublishResults,
		})
	}
	
	// Ensure consistent sorting by applying Go-level sort as backup
	// This guarantees the same order regardless of database query plan changes
	switch sortColumn {
	case "submitted_at":
		if sortOrder == "asc" {
			sort.Slice(results, func(i, j int) bool {
				return results[i].SubmittedAt < results[j].SubmittedAt
			})
		} else {
			sort.Slice(results, func(i, j int) bool {
				return results[i].SubmittedAt > results[j].SubmittedAt
			})
		}
	case "research_title":
		if sortOrder == "asc" {
			sort.Slice(results, func(i, j int) bool {
				if results[i].ResearchTitle == results[j].ResearchTitle {
					return results[i].SubmittedAt < results[j].SubmittedAt
				}
				return results[i].ResearchTitle < results[j].ResearchTitle
			})
		} else {
			sort.Slice(results, func(i, j int) bool {
				if results[i].ResearchTitle == results[j].ResearchTitle {
					return results[i].SubmittedAt > results[j].SubmittedAt
				}
				return results[i].ResearchTitle > results[j].ResearchTitle
			})
		}
	case "econsent_title":
		if sortOrder == "asc" {
			sort.Slice(results, func(i, j int) bool {
				if results[i].EconsentTitle == results[j].EconsentTitle {
					return results[i].SubmittedAt < results[j].SubmittedAt
				}
				return results[i].EconsentTitle < results[j].EconsentTitle
			})
		} else {
			sort.Slice(results, func(i, j int) bool {
				if results[i].EconsentTitle == results[j].EconsentTitle {
					return results[i].SubmittedAt > results[j].SubmittedAt
				}
				return results[i].EconsentTitle > results[j].EconsentTitle
			})
		}
	case "patient_name":
		if sortOrder == "asc" {
			sort.Slice(results, func(i, j int) bool {
				if results[i].PatientName == results[j].PatientName {
					return results[i].SubmittedAt < results[j].SubmittedAt
				}
				return results[i].PatientName < results[j].PatientName
			})
		} else {
			sort.Slice(results, func(i, j int) bool {
				if results[i].PatientName == results[j].PatientName {
					return results[i].SubmittedAt > results[j].SubmittedAt
				}
				return results[i].PatientName > results[j].PatientName
			})
		}
	}

	// 8. Calculate total pages
	totalPages := int((totalRecords + int64(perPage) - 1) / int64(perPage))

	response := GenomicResultListResponse[PatientResult]{
		Page:         page,
		PerPage:      perPage,
		Sort:         sortOrder,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      results,
	}
	// 9. Return response
	utils.JSONResponse(c, http.StatusOK, "Data fetched successfully!", response)
}

// UploadStudyResult allows a nurse to upload/publish a study result document
func UploadGenomicResult(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can upload genomic results", nil)
		return
	}

	patientIDStr := c.PostForm("patient_id")
	econsentIDStr := c.PostForm("eonsent_form_id")
	file, err := c.FormFile("genomic_result_file")
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Missing or invalid result_file", nil)
		return
	}
	if patientIDStr == "" || econsentIDStr == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Missing patient_id or econsent_form_id", nil)
		return
	}

	patientID, err := strconv.Atoi(patientIDStr)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid patient_id", nil)
		return
	}
	econsent_form_Id, err := strconv.Atoi(econsentIDStr)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid econsent_form_id", nil)
		return
	}

	// Validate assignment - allow approved, withdrawn, reviewed, and archived consents
	// Archived consents are valid for genomic result uploads as they represent historical participation
	var assignment models.EConsentAssignment
	err = config.DB.Preload("EConsentForm.Research").Where("patient_id = ? AND e_consent_form_id = ? AND status IN (?, ?, ?, ?) AND is_deleted = ?", 
		patientID, econsent_form_Id, models.EConsentStatusApproved, models.EConsentStatusWithdrawn, models.EConsentStatusReviewed, models.EConsentStatusArchived, false).First(&assignment).Error
	if err != nil {	
		utils.JSONResponse(c, http.StatusBadRequest, "Patient does not have a valid consent for this study", nil)
		return
	}
	
	// Check if consent is withdrawn or reviewed - warn but allow upload
	if assignment.Status == models.EConsentStatusWithdrawn || assignment.Status == models.EConsentStatusReviewed {
		// Log warning but continue - this allows uploading results for patients who withdrew after initial approval
		fmt.Printf("WARNING: Uploading genomic result for patient with %s consent status\n", assignment.Status)
	}

	// Verify that we have the research ID
	researchID := assignment.EConsentForm.ResearchID
	if researchID == 0 {
		// Fallback: Get research ID directly from e-consent form
		var econsentForm models.EConsentForm
		err = config.DB.Where("id = ?", econsent_form_Id).First(&econsentForm).Error
		if err != nil {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to get e-consent form details", nil)
			return
		}
		researchID = econsentForm.ResearchID
		if researchID == 0 {
			utils.JSONResponse(c, http.StatusInternalServerError, "Invalid e-consent form: missing research ID", nil)
			return
		}
	}

	// Open file and calculate hash first
	src, err := file.Open()
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to open uploaded file", nil)
		return
	}
	defer src.Close()

	h := sha256.New()
	if _, err := io.Copy(h, src); err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to hash uploaded file", nil)
		return
	}
	fileHash := fmt.Sprintf("%x", h.Sum(nil))

	// Check for existing record with same hash (anywhere in the system)
	var existing models.Genomics
	err = config.DB.Where("file_hash = ? AND is_deleted = ?", fileHash, false).First(&existing).Error
	if err == nil {
		utils.JSONResponse(c, http.StatusBadRequest, "This exact file has already been uploaded to the system", nil)
		return
	}

	// Rewind file since we need to re-read it after hash
	src.Seek(0, io.SeekStart)

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
		utils.JSONResponse(c, http.StatusBadRequest, "Only PDF or DOC files are allowed", nil)
		return
	}

	// Ensure the directory exists
	uploadDir := "public/genomic-result"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		fmt.Printf("DEBUG: Directory does not exist, creating: %s\n", uploadDir)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			fmt.Printf("DEBUG: Failed to create directory: %v\n", err)
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create upload directory", nil)
			return
		}
	}

	// Save to disk
	filename := fmt.Sprintf("result_patient%d_study%d_%d%s", patientID, econsent_form_Id, time.Now().UnixNano(), ext)
	filePath := filepath.Join("public/genomic-result", filename)
	
	// Debug: Log file creation attempt
	fmt.Printf("DEBUG: Attempting to create file at: %s\n", filePath)
	
	dst, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("DEBUG: File creation error: %v\n", err)
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save file to server", nil)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		fmt.Printf("DEBUG: File write error: %v\n", err)
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to write file", nil)
		return
	}
	
	// Debug: Log successful file creation
	fmt.Printf("DEBUG: File created successfully: %s\n", filePath)
	fileURL := "/public/genomic-result/" + filename

	// Get the current active consent assignment to determine the version
	var currentAssignment models.EConsentAssignment
	err = config.DB.Where("patient_id = ? AND e_consent_form_id = ? AND is_deleted = ?", 
		patientID, econsent_form_Id, false).
		Order("assigned_at desc").First(&currentAssignment).Error
	
	var versionID *uint
	if err == nil && currentAssignment.EConsentFormVersionID != 0 {
		versionID = &currentAssignment.EConsentFormVersionID
	}
	
	// Save in DB
	genomics := models.Genomics{
		PatientID:             uint(patientID),
		EConsentFormID:        uint(econsent_form_Id), 
		EConsentFormVersionID: versionID, // Link to the current consent version
		ResearchID:            uint(researchID),
		FileURL:               fileURL,
		FileHash:              fileHash,
		PublishedBy:           user.ID,
	}
	
	// Debug: Log the values being inserted
	fmt.Printf("DEBUG: Inserting genomics record - PatientID: %d, EConsentFormID: %d, ResearchID: %d\n", 
		genomics.PatientID, genomics.EConsentFormID, genomics.ResearchID)
	
	if err := config.DB.Create(&genomics).Error; err != nil {
		fmt.Printf("DEBUG: Database error: %v\n", err)
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save result metadata", nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, "Result uploaded successfully", gin.H{})
}

// EditGenomicResult allows a nurse to replace the genomic result document
func EditGenomicResult(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can edit genomic results", nil)
		return
	}

	// Get genomic result ID from URL parameter
	genomicIDStr := c.Param("id")
	if genomicIDStr == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Genomic result ID is required", nil)
		return
	}

	genomicID, err := strconv.Atoi(genomicIDStr)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid genomic result ID", nil)
		return
	}

	// Check if genomic result exists
	var existingGenomic models.Genomics
	err = config.DB.Where("id = ? AND is_deleted = ?", genomicID, false).First(&existingGenomic).Error
	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Genomic result not found", nil)
		return
	}

	// Check if genomic result is published - cannot edit published results
	if existingGenomic.IsPublished {
		utils.JSONResponse(c, http.StatusForbidden, "Cannot edit a published genomic result. Please unpublish it first.", nil)
		return
	}

	// Get the new file
	file, err := c.FormFile("genomic_result_file")
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Missing or invalid result_file", nil)
		return
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
		utils.JSONResponse(c, http.StatusBadRequest, "Only PDF or DOC files are allowed", nil)
		return
	}

	// Open file and calculate hash first
	src, err := file.Open()
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to open uploaded file", nil)
		return
	}
	defer src.Close()

	h := sha256.New()
	if _, err := io.Copy(h, src); err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to hash uploaded file", nil)
		return
	}
	fileHash := fmt.Sprintf("%x", h.Sum(nil))

	// Check for existing record with same hash (excluding current record)
	var duplicateGenomic models.Genomics
	err = config.DB.Where("file_hash = ? AND id != ? AND is_deleted = ?", fileHash, genomicID, false).First(&duplicateGenomic).Error
	if err == nil {
		utils.JSONResponse(c, http.StatusBadRequest, "This exact file has already been uploaded to the system", nil)
		return
	}

	// Rewind file since we need to re-read it after hash
	src.Seek(0, io.SeekStart)

	// Delete the old file
	oldFilePath := filepath.Join("public/genomic-result", filepath.Base(existingGenomic.FileURL))
	if _, err := os.Stat(oldFilePath); err == nil {
		os.Remove(oldFilePath)
	}

	// Save new file to disk
	filename := fmt.Sprintf("result_patient%d_study%d_%d%s", existingGenomic.PatientID, existingGenomic.EConsentFormID, time.Now().UnixNano(), ext)
	filePath := filepath.Join("public/genomic-result", filename)
	dst, err := os.Create(filePath)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save file to server", nil)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to write file", nil)
		return
	}
	fileURL := "/public/genomic-result/" + filename

	// Update in DB
	updates := map[string]interface{}{
		"file_url":     fileURL,
		"file_hash":    fileHash,
		"published_by": user.ID,
		"updated_at":   time.Now(),
	}

	if err := config.DB.Model(&models.Genomics{}).Where("id = ?", genomicID).Updates(updates).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update result metadata", nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, "Genomic result updated successfully", gin.H{
		"id":        genomicID,
		"file_url":  fileURL,
		"file_hash": fileHash,
	})
}

// DeleteGenomicResult allows a nurse to soft-delete a genomic result document
func DeleteGenomicResult(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can delete genomic results", nil)
		return
	}

	// Get genomic result ID from URL parameter
	genomicIDStr := c.Param("id")
	if genomicIDStr == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Genomic result ID is required", nil)
		return
	}

	genomicID, err := strconv.Atoi(genomicIDStr)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid genomic result ID", nil)
		return
	}

	// Check if genomic result exists
	var existingGenomic models.Genomics
	err = config.DB.Where("id = ? AND is_deleted = ?", genomicID, false).First(&existingGenomic).Error
	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Genomic result not found", nil)
		return
	}

	// Check if genomic result is published - cannot delete published results
	if existingGenomic.IsPublished {
		utils.JSONResponse(c, http.StatusForbidden, "Cannot delete a published genomic result. Please unpublish it first.", nil)
		return
	}

	// Soft delete the genomic result
	updates := map[string]interface{}{
		"is_deleted": true,
		"updated_at": time.Now(),
	}

	if err := config.DB.Model(&models.Genomics{}).Where("id = ?", genomicID).Updates(updates).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to delete genomic result", nil)
		return
	}

	// Optionally delete the physical file (uncomment if you want to delete the file too)
	// oldFilePath := filepath.Join("public/genomic-result", filepath.Base(existingGenomic.FileURL))
	// if _, err := os.Stat(oldFilePath); err == nil {
	// 	os.Remove(oldFilePath)
	// }

	utils.JSONResponse(c, http.StatusOK, "Genomic result deleted successfully", gin.H{
		"id": genomicID,
	})
}

// ListFilteredGenomicResults returns a paginated, filtered, and sorted list of genomic results
func ListGenomicResults(c *gin.Context) {
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "patient_id", "econsent_form_id"}
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid query parameters", nil)
		return
	}

	// Pagination
	page := 1
	if val := query.Get("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid page parameter", nil)
			return
		}
	}
	perPage := 10
	if val := query.Get("per_page"); val != "" {
		if pp, err := strconv.Atoi(val); err == nil && pp > 0 {
			perPage = pp
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid per_page parameter", nil)
			return
		}
	}

	// Sorting
	sort := "desc"
	if val := strings.ToLower(query.Get("sort")); val == "asc" || val == "desc" {
		sort = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid sort parameter", nil)
		return
	}
	sortColumn := "created_at"
	validSortColumns := []string{"created_at", "patient_name", "research_title", "econsent_title"}
	if val := strings.ToLower(query.Get("sort_column")); val != "" {
		if utils.IsInList(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid sort_column parameter", nil)
			return
		}
	}

	searchText := strings.TrimSpace(query.Get("search_text"))

	// Filter parameters
	patientID := strings.TrimSpace(query.Get("patient_id"))
	econsentFormID := strings.TrimSpace(query.Get("econsent_form_id"))

	// Build query
	db := config.DB.Model(&models.Genomics{}).
		Joins("JOIN users ON users.id = genomics.patient_id").
		Joins("JOIN e_consent_forms ON e_consent_forms.id = genomics.e_consent_form_id").
		Joins("JOIN researches ON researches.id = genomics.research_id").
		Where("genomics.is_deleted = ?", false)

	// Apply patient_id filter
	if patientID != "" {
		if pid, err := strconv.ParseUint(patientID, 10, 64); err == nil {
			db = db.Where("genomics.patient_id = ?", pid)
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid patient_id parameter", nil)
			return
		}
	}

	// Apply econsent_form_id filter
	if econsentFormID != "" {
		if eid, err := strconv.ParseUint(econsentFormID, 10, 64); err == nil {
			db = db.Where("genomics.e_consent_form_id = ?", eid)
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid econsent_form_id parameter", nil)
			return
		}
	}

	if searchText != "" {
		db = db.Where(
			"users.first_name ILIKE ? OR users.last_name ILIKE ? OR researches.title ILIKE ? OR e_consent_forms.title ILIKE ?",
			"%"+searchText+"%", "%"+searchText+"%", "%"+searchText+"%", "%"+searchText+"%",
		)
	}

	// Count total records
	var totalRecords int64
	db.Count(&totalRecords)

	// Map sort column to actual database columns
	var sortExpr string
	switch sortColumn {
	case "research_title":
		sortExpr = "researches.title " + sort
	case "econsent_title":
		sortExpr = "e_consent_forms.title " + sort
	case "created_at":
		fallthrough
	default:
		sortExpr = "genomics.created_at " + sort
	}

	// Pagination and sorting
	offset := (page - 1) * perPage
	db = db.Order(sortExpr).Limit(perPage).Offset(offset)

	// Fetch records
	var genomics []models.Genomics
	db.Preload("Patient").Preload("EConsentForm.Research").Find(&genomics)

	// Build response
	var results []GenomicResultResponse
	for _, g := range genomics {
		name := ""
		if g.Patient.ID != 0 {
		    name = strings.TrimSpace(g.Patient.FirstName + " " + g.Patient.LastName)
		}
		researchTitle := ""
		if g.EConsentForm.Research.ID != 0 {
			researchTitle = g.EConsentForm.Research.Title
		}
		econsentTitle := ""
		if g.EConsentForm.ID != 0 {
			econsentTitle = g.EConsentForm.Title
		}
		
		// Encrypt patient name
		encryptedName := ""
		if name != "" {
			encryptedName, _ = utils.Encrypt(name)
		}
		
		// Get current consent status for this patient and e-consent form
		var consentStatus string = "unknown"
		
		// First try to get from EConsentAssignment (most current status)
		var assignment models.EConsentAssignment
		if err := config.DB.Where("patient_id = ? AND e_consent_form_id = ? AND is_deleted = ?", 
			g.PatientID, g.EConsentFormID, false).
			Order("assigned_at desc").First(&assignment).Error; err == nil {
			consentStatus = string(assignment.Status)
		} else {
			// If no assignment found, try to get from PatientEConsent table
			var patientEConsent models.PatientEConsent
			if err := config.DB.Where("patient_id = ? AND e_consent_form_id = ? AND is_deleted = ?", 
				g.PatientID, g.EConsentFormID, false).
				Order("created_at desc").First(&patientEConsent).Error; err == nil {
				consentStatus = string(patientEConsent.Status)
			} else {
				// If still not found, check if there are any assignments at all
				var count int64
				if err := config.DB.Model(&models.EConsentAssignment{}).
					Where("patient_id = ? AND e_consent_form_id = ?", g.PatientID, g.EConsentFormID).
					Count(&count).Error; err == nil && count > 0 {
					consentStatus = "archived" // If assignments exist but none are active
				} else {
					consentStatus = "no_consent" // No consent records found
				}
			}
		}
		
		// Extract only filename from the full path
		filename := filepath.Base(g.FileURL)
		
		// Get version information
		var versionNumber *int
		if g.EConsentFormVersionID != nil {
			var version models.EConsentFormVersion
			if err := config.DB.Where("id = ?", *g.EConsentFormVersionID).First(&version).Error; err == nil {
				versionNumber = &version.VersionNumber
			}
		}
		
		results = append(results, GenomicResultResponse{
			ID:                    g.ID,
			PatientID:             g.PatientID,
			EConsentFormID:        g.EConsentFormID,
			EConsentFormVersionID: g.EConsentFormVersionID,
			VersionNumber:         versionNumber,
			PatientName:           encryptedName,
			ResearchTitle:         researchTitle,
			EConsentTitle:         econsentTitle,
			FileURL:               filename,
			IsPublished:           g.IsPublished,
			CreatedAt:             g.CreatedAt.Format(time.RFC3339),
			ConsentStatus:         consentStatus,
		})
	}

	totalPages := int((totalRecords + int64(perPage) - 1) / int64(perPage))

	response := GenomicResultListResponse[GenomicResultResponse]{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      results,
	}
	utils.JSONResponse(c, http.StatusOK, "Results fetched successfully!", response)
}

// ServeGenomicResultFile serves the uploaded genomic result file
func ServeGenomicResultFile(c *gin.Context) {
	// Get the filename from the URL parameter
	filename := c.Param("filename")
	if filename == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Filename is required", nil)
		return
	}

	// Construct the file path
	filePath := filepath.Join("public/genomic-result", filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		utils.JSONResponse(c, http.StatusNotFound, "File not found", nil)
		return
	}

	// Get file info for content type
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to get file info", nil)
		return
	}

	// Determine content type based on file extension
	ext := strings.ToLower(filepath.Ext(filename))
	var contentType string
	switch ext {
	case ".pdf":
		contentType = "application/pdf"
	case ".doc":
		contentType = "application/msword"
	case ".docx":
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		contentType = "application/octet-stream"
	}

	// Set headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Serve the file
	c.File(filePath)
}

// DownloadGenomicResultFile allows downloading the genomic result file
func DownloadGenomicResultFile(c *gin.Context) {
	// Get the filename from the URL parameter
	filename := c.Param("filename")
	if filename == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Filename is required", nil)
		return
	}

	// Construct the file path
	filePath := filepath.Join("public/genomic-result", filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		utils.JSONResponse(c, http.StatusNotFound, "File not found", nil)
		return
	}

	// Get file info for content type
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to get file info", nil)
		return
	}

	// Determine content type based on file extension
	ext := strings.ToLower(filepath.Ext(filename))
	var contentType string
	switch ext {
	case ".pdf":
		contentType = "application/pdf"
	case ".doc":
		contentType = "application/msword"
	case ".docx":
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		contentType = "application/octet-stream"
	}

	// Set headers for download
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Cache-Control", "no-cache")

	// Serve the file
	c.File(filePath)
}


// PublishGenomicResult allows a nurse to publish/unpublish a genomic result
func PublishGenomicResult(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can publish genomic results", nil)
		return
	}

	// Get genomic result ID from URL parameter
	genomicIDStr := c.Param("id")
	if genomicIDStr == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Genomic result ID is required", nil)
		return
	}

	genomicID, err := strconv.Atoi(genomicIDStr)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid genomic result ID", nil)
		return
	}

	// Get publish status from request body
	var requestBody struct {
		IsPublished bool `json:"is_published" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid request body. is_published field is required", nil)
		return
	}

	// Check if genomic result exists and get related data
	var existingGenomic models.Genomics
	err = config.DB.Preload("Patient").Preload("EConsentForm.Research").
		Where("id = ? AND is_deleted = ?", genomicID, false).First(&existingGenomic).Error
	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Genomic result not found", nil)
		return
	}
	
	// Check current consent status for this patient and e-consent form
	var assignment models.EConsentAssignment
	err = config.DB.Where("patient_id = ? AND e_consent_form_id = ? AND is_deleted = ?", 
		existingGenomic.PatientID, existingGenomic.EConsentFormID, false).
		Order("assigned_at desc").First(&assignment).Error
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Patient consent assignment not found", nil)
		return
	}
	
	// Prevent publishing if consent is withdrawn or reviewed
	if requestBody.IsPublished && (assignment.Status == models.EConsentStatusWithdrawn || assignment.Status == models.EConsentStatusReviewed) {
		utils.JSONResponse(c, http.StatusBadRequest, "Cannot publish genomic results for patients with withdrawn or reviewed consent", nil)
		return
	}

	// Update publication status
	updates := map[string]interface{}{
		"is_published": requestBody.IsPublished,
		"updated_at":   time.Now(),
	}

	if err := config.DB.Model(&models.Genomics{}).Where("id = ?", genomicID).Updates(updates).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update publication status", nil)
		return
	}

	// If publishing (not unpublishing), send notification and email to patient
	if requestBody.IsPublished {
		// Send notification to patient
		notificationTitle := "Genomic Results Published"
		notificationMessage := fmt.Sprintf("Your genomic results for research '%s' have been published and are now available for review.", existingGenomic.EConsentForm.Research.Title)
		notificationType := "Genomic-Results"
		entityType := "genomics"
		entityID := existingGenomic.ID
		username := user.FirstName + " " + user.LastName
		metadata := map[string]interface{}{
			"genomic_id":      existingGenomic.ID,
			"research_id":     existingGenomic.EConsentForm.ResearchID,
			"econsent_form_id": existingGenomic.EConsentFormID,
			"status":          "published",
		}

		err = services.CreateNotification(
			config.DB,
			existingGenomic.PatientID,
			notificationTitle,
			notificationMessage,
			notificationType,
			username,
			entityType,
			entityID,
			metadata,
		)
		if err != nil {
			// Log but don't fail the main response
			fmt.Printf("Failed to create notification for patient %d: %v\n", existingGenomic.PatientID, err)
		}

		// Send email to patient
		if existingGenomic.Patient.ID != 0 && existingGenomic.Patient.Email != "" {
			// Construct result link (this should point to where patients can view their results)
			resultLink := fmt.Sprintf("%s/patient/genomic-results/%d", config.AppConfig.AppUrl, existingGenomic.ID)
			
			emailBody := emails.GenomicResultPublishedEmail(
				existingGenomic.Patient.FirstName,
				existingGenomic.Patient.LastName,
				existingGenomic.EConsentForm.Research.Title,
				existingGenomic.EConsentForm.Title,
				resultLink,
			)
			subject := "Your Genomic Results Are Ready"
			
			err = config.SendEmail([]string{existingGenomic.Patient.Email}, subject, emailBody)
			if err != nil {
				// Log but don't fail the main response
				fmt.Printf("Failed to send email to patient %d: %v\n", existingGenomic.PatientID, err)
			}
		}
	}

	statusText := "published"
	if !requestBody.IsPublished {
		statusText = "unpublished"
	}

	utils.JSONResponse(c, http.StatusOK, fmt.Sprintf("Genomic result %s successfully", statusText), gin.H{
		"id":           genomicID,
		"is_published": requestBody.IsPublished,
	})
}

// GetPatientGenomicResults allows patients to view their own genomic results
func GetPatientGenomicResults(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Only allow patients to access this endpoint
	if user.RoleID != 4 {
		utils.JSONResponse(c, http.StatusForbidden, "Only patients can view their genomic results", nil)
		return
	}

	// Define allowed query parameters for filtering and pagination
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "econsent_form_id"}
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

	// Parse and validate 'sort_column' parameter (default: created_at)
	sortColumn := "created_at"
	validSortColumns := []string{"created_at", "research_title", "econsent_title"}
	if val := query.Get("sort_column"); val != "" {
		if utils.StringInSlice(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortColumnParameter, nil)
			return
		}
	}

	// Get the search text for filtering (optional)
	searchText := query.Get("search_text")

	// Get the econsent_form_id filter (optional)
	econsentFormID := query.Get("econsent_form_id")

	// Build the base GORM query for Genomics, joining with related tables
	// Only show results for the authenticated patient
	db := config.DB.Model(&models.Genomics{}).
		Where("genomics.patient_id = ? AND genomics.is_deleted = ?", user.ID, false).
		Where("genomics.is_published = ?", true).
		Joins("LEFT JOIN e_consent_forms ON genomics.e_consent_form_id = e_consent_forms.id").
		Joins("LEFT JOIN researches ON e_consent_forms.research_id = researches.id")

	// Apply econsent_form_id filter if provided
	if econsentFormID != "" {
		if eid, err := strconv.ParseUint(econsentFormID, 10, 64); err == nil {
			db = db.Where("genomics.e_consent_form_id = ?", eid)
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid econsent_form_id parameter", nil)
			return
		}
	}

	// If search text is provided, filter by research title or econsent title
	if searchText != "" {
		searchPattern := "%" + searchText + "%"
		db = db.Where(
			"researches.title ILIKE ? OR e_consent_forms.title ILIKE ?",
			searchPattern, searchPattern,
		)
	}

	// Count total records for pagination
	var totalRecords int64
	if err := db.Count(&totalRecords).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCountRecords, nil)
		return
	}
	
	// Debug: Log the count result
	fmt.Printf("DEBUG: Found %d total genomic results for patient %d\n", totalRecords, user.ID)

	// Calculate total pages
	var totalPages int
	if totalRecords == 0 {
		totalPages = 0
	} else {
		totalPages = int((totalRecords + int64(perPage) - 1) / int64(perPage))
	}

	// Map sortColumn to actual DB columns or expressions
	var sortExpr string
	switch sortColumn {
	case "research_title":
		sortExpr = "researches.title " + sort
	case "econsent_title":
		sortExpr = "e_consent_forms.title " + sort
	case "created_at":
		fallthrough
	default:
		sortExpr = "genomics.created_at " + sort
	}

	// Apply sorting and pagination
	db = db.Order(sortExpr)
	offset := (page - 1) * perPage
	db = db.Limit(perPage).Offset(offset)

	// Fetch the genomic results with related data
	var genomics []models.Genomics
	if err := db.Preload("EConsentForm.Research").Find(&genomics).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Prepare the response records
	var responses []GenomicResultResponse
	for _, genomic := range genomics {
		researchTitle := ""
		if genomic.EConsentForm.Research.ID != 0 {
			researchTitle = genomic.EConsentForm.Research.Title
		}
		econsentTitle := ""
		if genomic.EConsentForm.ID != 0 {
			econsentTitle = genomic.EConsentForm.Title
		}
		
		// Get current consent status for this patient and e-consent form
		var consentStatus string = "unknown"
		
		// First try to get from EConsentAssignment (most current status)
		var assignment models.EConsentAssignment
		if err := config.DB.Where("patient_id = ? AND e_consent_form_id = ? AND is_deleted = ?", 
			genomic.PatientID, genomic.EConsentFormID, false).
			Order("assigned_at desc").First(&assignment).Error; err == nil {
			consentStatus = string(assignment.Status)
		} else {
			// If no assignment found, try to get from PatientEConsent table
			var patientEConsent models.PatientEConsent
			if err := config.DB.Where("patient_id = ? AND e_consent_form_id = ? AND is_deleted = ?", 
				genomic.PatientID, genomic.EConsentFormID, false).
				Order("created_at desc").First(&patientEConsent).Error; err == nil {
				consentStatus = string(patientEConsent.Status)
			} else {
				// If still not found, check if there are any assignments at all
				var count int64
				if err := config.DB.Model(&models.EConsentAssignment{}).
					Where("patient_id = ? AND e_consent_form_id = ?", genomic.PatientID, genomic.EConsentFormID).
					Count(&count).Error; err == nil && count > 0 {
					consentStatus = "archived" // If assignments exist but none are active
				} else {
					consentStatus = "no_consent" // No consent records found
				}
			}
		}
		
		// Extract only filename from the full path
		filename := filepath.Base(genomic.FileURL)
		
		// Get version information
		var versionNumber *int
		if genomic.EConsentFormVersionID != nil {
			var version models.EConsentFormVersion
			if err := config.DB.Where("id = ?", *genomic.EConsentFormVersionID).First(&version).Error; err == nil {
				versionNumber = &version.VersionNumber
			}
		}
		
		response := GenomicResultResponse{
			ID:                    genomic.ID,
			PatientID:             genomic.PatientID,
			EConsentFormID:        genomic.EConsentFormID,
			EConsentFormVersionID: genomic.EConsentFormVersionID,
			VersionNumber:         versionNumber,
			// PatientName:    "", // Don't include patient name in patient view
			ResearchTitle:  researchTitle,
			EConsentTitle:  econsentTitle,
			FileURL:        filename,
			IsPublished:    genomic.IsPublished,
			CreatedAt:      genomic.CreatedAt.Format(time.RFC3339),
			ConsentStatus:  consentStatus,
		}

		responses = append(responses, response)
	}

	// Prepare the paginated response structure
	response := GenomicResultListResponse[GenomicResultResponse]{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      responses,
	}

	utils.JSONResponse(c, http.StatusOK, "Patient genomic results fetched successfully", response)
}