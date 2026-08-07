package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	MaxResearchDocSize = 10 * 1024 * 1024 // 10MB
	AllowedDocTypes    = "application/pdf"
	ResearchDocPath    = "public/research_documents"
)

// FileDuplicateInfo represents information about a duplicate file
type FileDuplicateInfo struct {
	ResearchID    uint   `json:"research_id"`
	ResearchTitle string `json:"research_title"`
	IsPublished   bool   `json:"is_published"`
	FileHash      string `json:"file_hash"`
}

// CreateResearchRequest represents the request body for creating a research
type CreateResearchRequest struct {
	Title       string `form:"title" binding:"required,min=3,max=30"`
	Description string `form:"description" binding:"omitempty,min=1,max=100"`
	CategoryID  uint   `form:"category_id" binding:"required"`
	IsPublished bool   `form:"is_published"`
}

// ResearchResponse represents a single research in the response
type ResearchResponse struct {
	ID           uint                      `json:"id"`
	Title        string                    `json:"title"`
	Description  string                    `json:"description"`
	CategoryID   uint                      `json:"category_id"`
	CategoryName string                    `json:"category_name"`
	CreatedBy    uint                      `json:"created_by"`
	UserName     string                    `json:"user_name"`
	CreatedAt    time.Time                 `json:"created_at"`
	UpdatedAt    time.Time                 `json:"updated_at"`
	IsPublished  bool                      `json:"is_published"`
	Document     *ResearchDocumentResponse `json:"document,omitempty"`
}

// ResearchDocumentResponse represents a research document in the response
type ResearchDocumentResponse struct {
	ID         uint      `json:"id"`
	FilePath   string    `json:"file_path"`
	FileType   string    `json:"file_type"`
	UploadedBy uint      `json:"uploaded_by"`
	UserName   string    `json:"user_name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ResearchesResponse represents the structured response for researches list
type ResearchesResponse struct {
	Page         int                `json:"page"`
	PerPage      int                `json:"per_page"`
	Sort         string             `json:"sort"`
	SortColumn   string             `json:"sort_column"`
	SearchText   string             `json:"search_text"`
	TotalRecords int64              `json:"total_records"`
	TotalPages   int                `json:"total_pages"`
	Records      []ResearchResponse `json:"records"`
}

// UpdateResearchRequest represents the request body for updating a research
type UpdateResearchRequest struct {
	Title       string `form:"title" binding:"omitempty,min=3,max=30"`
	Description string `form:"description" binding:"omitempty,min=1,max=100"`
	CategoryID  uint   `form:"category_id" binding:"omitempty"`
	IsPublished bool   `form:"is_published"`
}

// checkFileDuplicate checks if a file hash already exists in the database
func checkFileDuplicate(fileHash string, excludeResearchID uint) (*FileDuplicateInfo, error) {
	var existingDoc models.ResearchDocument
	var research models.Research

	query := config.DB.Joins("JOIN researches ON research_documents.research_id = researches.id").
		Where("research_documents.file_hash = ? AND research_documents.is_deleted = ? AND researches.is_deleted = ?",
			fileHash, false, false)

	if excludeResearchID > 0 {
		query = query.Where("research_documents.research_id != ?", excludeResearchID)
	}

	err := query.First(&existingDoc).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No duplicate found
		}
		return nil, err
	}

	// Get the research details
	if err := config.DB.Where("id = ?", existingDoc.ResearchID).First(&research).Error; err != nil {
		return nil, err
	}

	return &FileDuplicateInfo{
		ResearchID:    research.ID,
		ResearchTitle: research.Title,
		IsPublished:   research.IsPublished,
		FileHash:      existingDoc.FileHash,
	}, nil
}

// handleFileDuplicate handles the duplicate file scenario based on research status
func handleFileDuplicate(c *gin.Context, duplicateInfo *FileDuplicateInfo) {
	if duplicateInfo.IsPublished {
		// File is attached to a published research
		c.JSON(http.StatusConflict, gin.H{
			"error":   "File already exists",
			"message": fmt.Sprintf("This file is already attached to the published research '%s'. Please use a different file.", duplicateInfo.ResearchTitle),
		})
	} else {
		// File is attached to a draft research
		c.JSON(http.StatusConflict, gin.H{
			"error":   "File already exists",
			"message": fmt.Sprintf("This file is already attached to the draft research '%s'. Please upload a new file.", duplicateInfo.ResearchTitle),
		})
	}
}

// handleUpdateFileDuplicate handles the duplicate file scenario specifically for update operations
func handleUpdateFileDuplicate(c *gin.Context, duplicateInfo *FileDuplicateInfo) {
	c.JSON(http.StatusConflict, gin.H{
		"error":   "File already exists",
		"message": fmt.Sprintf("The file you're trying to upload already exists in another research. Please upload a new file", duplicateInfo.ResearchTitle),
	})
}

// CreateResearch creates a new research with an attached document
func CreateResearch(c *gin.Context) {
	// Get user from context using the middleware helper
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Only super-admin and admin users can create researches"})
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(MaxResearchDocSize); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Failed to parse form data"})
		return
	}

	// Bind form data
	var req CreateResearchRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Check if research with same title already exists in the same category
	var existingResearch models.Research
	if err := config.DB.Where("LOWER(title) = LOWER(?) AND category_id = ? AND is_deleted = ?", req.Title, req.CategoryID, false).First(&existingResearch).Error; err == nil {
		c.JSON(http.StatusConflict, ErrorResponse{Error: "A research with this title already exists in this category"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to check research title"})
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Document file is required"})
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > MaxResearchDocSize {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("File size exceeds maximum limit of %d MB", MaxResearchDocSize/1024/1024)})
		return
	}

	// Validate file type
	fileType := header.Header.Get("Content-Type")
	if !strings.Contains(AllowedDocTypes, fileType) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid file type. Only PDF-based documents are allowed"})
		return
	}

	// Calculate file hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to calculate file hash"})
		return
	}
	fileHash := hex.EncodeToString(hash.Sum(nil))

	// Reset file pointer for saving
	file.Seek(0, 0)

	// Check for file duplicate (for create, excludeResearchID is 0)
	duplicateInfo, err := checkFileDuplicate(fileHash, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to check file duplicate"})
		return
	}

	if duplicateInfo != nil {
		handleFileDuplicate(c, duplicateInfo)
		return
	}

	// Verify that the category exists and is not deleted
	var category models.ResearchCategory
	if err := config.DB.Where("id = ? AND is_deleted = ?", req.CategoryID, false).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid research category"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to validate research category"})
		return
	}

	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to start transaction"})
		return
	}

	// Create research record
	research := models.Research{
		Title:       req.Title,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		CreatedBy:   user.ID,
		IsPublished: req.IsPublished,
	}

	if err := tx.Create(&research).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create research"})
		return
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(ResearchDocPath, 0755); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create upload directory"})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d_%s%s", research.ID, utils.GenerateUniqueString(), ext)
	filepath := filepath.Join(ResearchDocPath, filename)

	// Save the file
	if err := c.SaveUploadedFile(header, filepath); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to save document"})
		return
	}

	// Create research document record
	doc := models.ResearchDocument{
		ResearchID: research.ID,
		FilePath:   filepath,
		FileType:   fileType,
		FileHash:   fileHash,
		UploadedBy: user.ID,
	}

	if err := tx.Create(&doc).Error; err != nil {
		tx.Rollback()
		// Clean up the uploaded file
		os.Remove(filepath)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create research document"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		// Clean up the uploaded file
		os.Remove(filepath)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to commit transaction"})
		return
	}

	// Get the user's name for the response
	var userName string
	if err := config.DB.Model(&models.User{}).
		Select("CONCAT(first_name, ' ', COALESCE(last_name, ''))").
		Where("id = ?", user.ID).
		Scan(&userName).Error; err != nil {
		userName = "" // Set empty if we can't get the name
	}

	// Prepare document response
	docResponse := ResearchDocumentResponse{
		ID:         doc.ID,
		FilePath:   doc.FilePath,
		FileType:   doc.FileType,
		UploadedBy: doc.UploadedBy,
		UserName:   userName,
		CreatedAt:  doc.CreatedAt,
		UpdatedAt:  doc.UpdatedAt,
	}

	response := ResearchResponse{
		ID:           research.ID,
		Title:        research.Title,
		Description:  research.Description,
		CategoryID:   research.CategoryID,
		CategoryName: category.Name,
		CreatedBy:    research.CreatedBy,
		UserName:     userName,
		CreatedAt:    research.CreatedAt,
		UpdatedAt:    research.UpdatedAt,
		IsPublished:  research.IsPublished,
		Document:     &docResponse,
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Research created successfully with document",
		"data":    response,
	})
}

// ListResearches retrieves a list of researches with filtering, pagination, and sorting
func ListResearches(c *gin.Context) {
	// Check authorization
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 && user.RoleID != 4 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin, admin, or patient users can view researches", nil)
		return
	}

	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "category_id", "status"}

	// Parse query parameters with default values
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidQueryParameters, nil)
		return
	}

	// Default and validation for 'status'
	status := "all"
	if val := strings.ToLower(query.Get("status")); val != "" {
		validStatuses := []string{"all", "draft", "published"}
		if !utils.StringInSlice(val, validStatuses) {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid status parameter. Must be one of: all, draft, published", nil)
			return
		}
		status = val
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
	validSortColumns := []string{
		"id", "title", "description", "category_id",
		"created_by", "created_at", "updated_at", "is_published",
		"category_name", "user_name",
	}
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

	// Parse search text for date patterns using the utility function
	dateResult := utils.ParseSearchTextForDate(searchText)

	// Optional 'category_id' filter
	var categoryID uint
	if val := query.Get("category_id"); val != "" {
		if id, err := strconv.ParseUint(val, 10, 32); err == nil {
			categoryID = uint(id)
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid category_id parameter", nil)
			return
		}
	}

	// Build query
	db := config.DB.Model(&models.Research{}).
		Select("researches.*, "+
			"research_categories.name as category_name, "+
			"CONCAT(users.first_name, ' ', COALESCE(users.last_name, '')) as user_name").
		Joins("LEFT JOIN research_categories ON researches.category_id = research_categories.id").
		Joins("LEFT JOIN users ON researches.created_by = users.id").
		Where("researches.is_deleted = ?", false)

	// Debug: Check total researches in database (before any search filter)
	var totalResearchesInDB int64
	config.DB.Model(&models.Research{}).Where("is_deleted = ?", false).Count(&totalResearchesInDB)

	// If patient, only show published researches
	if user.RoleID == 4 {
		db = db.Where("researches.is_published = ?", true)
	}

	// Apply status filter if provided
	if status != "all" {
		if status == "published" {
			db = db.Where("researches.is_published = ?", true)
		} else if status == "draft" {
			db = db.Where("researches.is_published = ?", false)
		}
	}

	// Apply category filter if provided
	if categoryID > 0 {
		db = db.Where("researches.category_id = ?", categoryID)
	}

	// Apply search filter if provided
	if searchText != "" {
		// Build comprehensive search query for all text fields including joined tables
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
				dateConditions = append(dateConditions, "(researches.created_at >= ? AND researches.created_at < ?)")
				dateArgs = append(dateArgs, startOfDay, endOfDay)
			} else if len(dateResult.Dates) > 0 {
				// Multiple dates - create OR conditions for each date
				for _, date := range dateResult.Dates {
					startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
					endOfDay := startOfDay.Add(24 * time.Hour)
					dateConditions = append(dateConditions, "(researches.created_at >= ? AND researches.created_at < ?)")
					dateArgs = append(dateArgs, startOfDay, endOfDay)
				}
			}

			// Build the combined search query
			combinedSearch := `
			(
				(LOWER(researches.title) LIKE ? OR
				LOWER(research_categories.name) LIKE ?
				)
				`

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
				(LOWER(researches.title) LIKE ? OR
				LOWER(research_categories.name) LIKE ? OR
				EXTRACT(DAY FROM researches.created_at) = ?)
			`
			db = db.Where(combinedSearch, searchPattern, searchPattern, *dateResult.Day)
		} else {
			// Text search only
			combinedSearch := `
			(
				LOWER(researches.title) LIKE ? OR
				LOWER(research_categories.name) LIKE ?
		)
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

	// Apply sorting with special handling for joined fields
	if sortColumn == "category_name" {
		db = db.Order("research_categories.name " + sort)
	} else if sortColumn == "user_name" {
		db = db.Order("CONCAT(users.first_name, ' ', COALESCE(users.last_name, '')) " + sort)
	} else {
		db = db.Order("researches." + sortColumn + " " + sort)
	}

	// Apply pagination
	offset := (page - 1) * perPage
	var researches []struct {
		models.Research
		CategoryName string `gorm:"column:category_name"`
		UserName     string `gorm:"column:user_name"`
	}
	if err := db.Limit(perPage).Offset(offset).Find(&researches).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Get document information for each research
	records := make([]ResearchResponse, len(researches))
	for i, r := range researches {
		// Get document information
		var doc models.ResearchDocument
		var docResponse *ResearchDocumentResponse
		if err := config.DB.Where("research_id = ? AND is_deleted = ?", r.ID, false).First(&doc).Error; err == nil {
			docResponse = &ResearchDocumentResponse{
				ID:         doc.ID,
				FilePath:   doc.FilePath,
				FileType:   doc.FileType,
				UploadedBy: doc.UploadedBy,
				UserName:   r.UserName,
				CreatedAt:  doc.CreatedAt,
				UpdatedAt:  doc.UpdatedAt,
			}
		}

		records[i] = ResearchResponse{
			ID:           r.ID,
			Title:        r.Title,
			Description:  r.Description,
			CategoryID:   r.CategoryID,
			CategoryName: r.CategoryName,
			CreatedBy:    r.CreatedBy,
			UserName:     r.UserName,
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
			IsPublished:  r.IsPublished,
			Document:     docResponse,
		}
	}

	response := ResearchesResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      records,
	}

	utils.JSONResponse(c, http.StatusOK, "Researches fetched successfully", response)
}

// DeleteResearch soft deletes a research and its associated document
func DeleteResearch(c *gin.Context) {
	// Get user from context using the middleware helper
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can delete researches", nil)
		return
	}

	// Get research ID from URL parameter
	researchID := c.Param("id")
	if researchID == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Research ID is required", nil)
		return
	}

	// Convert researchID to uint
	id, err := strconv.ParseUint(researchID, 10, 32)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid research ID format", nil)
		return
	}

	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Check if research exists and is not deleted
	var research models.Research
	if err := tx.Where("id = ? AND is_deleted = ?", id, false).First(&research).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusNotFound, "Research not found or already deleted", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch research: %v", err), nil)
		}
		return
	}

	// Check if research is published
	if research.IsPublished {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusForbidden, "Cannot delete a published research.", nil)
		return
	}

	// Soft delete the research
	now := time.Now()
	updates := map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
		"deleted_at": now,
	}

	if err := tx.Model(&models.Research{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to delete research: %v", err), nil)
		return
	}

	// Soft delete associated document if exists
	if err := tx.Model(&models.ResearchDocument{}).
		Where("research_id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": now,
			"deleted_at": now,
		}).Error; err != nil && err != gorm.ErrRecordNotFound {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to delete research document: %v", err), nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to commit transaction: %v", err), nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, "Research deleted successfully", nil)
}

// GetResearchDocument serves a research document file
func GetResearchDocument(c *gin.Context) {
	// Get user from context using the middleware helper
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 && user.RoleID != 4 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin, admin and patients users can view research documents", nil)
		return
	}

	// Get research ID from URL parameter
	researchID := c.Param("id")
	if researchID == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Research ID is required", nil)
		return
	}

	// Convert researchID to uint
	id, err := strconv.ParseUint(researchID, 10, 32)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid research ID format", nil)
		return
	}

	// Get research document
	var doc models.ResearchDocument
	if err := config.DB.
		Joins("JOIN researches ON research_documents.research_id = researches.id").
		Where("research_documents.research_id = ? AND research_documents.is_deleted = ? AND researches.is_deleted = ?", id, false, false).
		First(&doc).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusNotFound, "Research document not found", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch research document: %v", err), nil)
		}
		return
	}

	// Check if file exists
	if _, err := os.Stat(doc.FilePath); os.IsNotExist(err) {
		utils.JSONResponse(c, http.StatusNotFound, "Document file not found on server", nil)
		return
	}

	// Get file info for content length
	fileInfo, err := os.Stat(doc.FilePath)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to get file information", nil)
		return
	}

	// Set appropriate headers
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", filepath.Base(doc.FilePath)))
	c.Header("Content-Type", doc.FileType)
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	// Serve the file
	c.File(doc.FilePath)
}

// UpdateResearch updates an existing research and optionally its document
func UpdateResearch(c *gin.Context) {
	// Get user from context using the middleware helper
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can update researches", nil)
		return
	}

	// Get research ID from URL parameter
	researchID := c.Param("id")
	if researchID == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Research ID is required", nil)
		return
	}

	// Convert researchID to uint
	id, err := strconv.ParseUint(researchID, 10, 32)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid research ID format", nil)
		return
	}

	// Parse multipart form if it's a multipart request
	if c.Request.Header.Get("Content-Type") == "multipart/form-data" {
		if err := c.Request.ParseMultipartForm(MaxResearchDocSize); err != nil {
			utils.JSONResponse(c, http.StatusBadRequest, "Failed to parse form data", nil)
			return
		}
	}

	// Bind form data
	var req UpdateResearchRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Check if research exists and is not deleted
	var research models.Research
	if err := tx.Where("id = ? AND is_deleted = ?", id, false).First(&research).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusNotFound, "Research not found", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch research: %v", err), nil)
		}
		return
	}

	// If title is being updated, check for duplicates within the same category
	if req.Title != "" && req.Title != research.Title {
		// Determine which category to check against
		targetCategoryID := research.CategoryID
		if req.CategoryID != 0 {
			targetCategoryID = req.CategoryID
		}

		var count int64
		if err := tx.Model(&models.Research{}).
			Where("LOWER(title) = LOWER(?) AND category_id = ? AND id != ? AND is_deleted = ?", req.Title, targetCategoryID, id, false).
			Count(&count).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to check title uniqueness", nil)
			return
		}
		if count > 0 {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusConflict, "A research with this title already exists in this category", nil)
			return
		}
	}

	// If category is being updated, verify it exists
	if req.CategoryID != 0 && req.CategoryID != research.CategoryID {
		var category models.ResearchCategory
		if err := tx.Where("id = ? AND is_deleted = ?", req.CategoryID, false).First(&category).Error; err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid research category", nil)
			} else {
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to validate research category", nil)
			}
			return
		}
	}

	// Update research fields
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.CategoryID != 0 {
		updates["category_id"] = req.CategoryID
	}
	updates["is_published"] = req.IsPublished

	// Update research
	if err := tx.Model(&research).Updates(updates).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to update research: %v", err), nil)
		return
	}

	// Handle document update if a new file is provided
	file, header, err := c.Request.FormFile("document")
	if err == nil {
		defer file.Close()

		// Validate file size
		if header.Size > MaxResearchDocSize {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, fmt.Sprintf("File size exceeds maximum limit of %d MB", MaxResearchDocSize/1024/1024), nil)
			return
		}

		// Validate file type
		fileType := header.Header.Get("Content-Type")
		if !strings.Contains(AllowedDocTypes, fileType) {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid file type. Only PDF-based documents are allowed", nil)
			return
		}

		// Get existing document
		var existingDoc models.ResearchDocument
		if err := tx.Where("research_id = ? AND is_deleted = ?", id, false).First(&existingDoc).Error; err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch existing document", nil)
			return
		}

		// Calculate hash of the new file
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to calculate file hash", nil)
			return
		}
		newFileHash := hex.EncodeToString(hash.Sum(nil))

		// Check for file duplicate (exclude current research ID)
		duplicateInfo, err := checkFileDuplicate(newFileHash, uint(id))
		if err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to check file duplicate", nil)
			return
		}

		if duplicateInfo != nil {
			tx.Rollback()
			handleUpdateFileDuplicate(c, duplicateInfo)
			return
		}

		// If there's an existing document, compare hashes
		if existingDoc.ID != 0 {
			// If hashes match, skip the document update but continue with other updates
			if existingDoc.FileHash == newFileHash {
				// Reset file pointer for potential future use
				file.Seek(0, 0)
				// Continue with the transaction - document unchanged but other fields may be updated
				// Don't return early, let the transaction continue to handle title/description updates
			} else {
				// Hashes don't match, proceed with document replacement
				// Create directory if it doesn't exist
				if err := os.MkdirAll(ResearchDocPath, 0755); err != nil {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create upload directory", nil)
					return
				}

				// Generate unique filename
				ext := filepath.Ext(header.Filename)
				filename := fmt.Sprintf("%d_%s%s", id, utils.GenerateUniqueString(), ext)
				filepath := filepath.Join(ResearchDocPath, filename)

				// Reset file pointer before saving
				file.Seek(0, 0)

				// Save the new file
				if err := c.SaveUploadedFile(header, filepath); err != nil {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save document", nil)
					return
				}

				// Mark existing document as deleted
				now := time.Now()
				if err := tx.Model(&models.ResearchDocument{}).
					Where("id = ?", existingDoc.ID).
					Updates(map[string]interface{}{
						"is_deleted": true,
						"updated_at": now,
						"deleted_at": now,
					}).Error; err != nil {
					tx.Rollback()
					// Clean up the new file
					os.Remove(filepath)
					utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update existing document", nil)
					return
				}

				// Delete the old file
				if err := os.Remove(existingDoc.FilePath); err != nil {
					// Log the error but continue since the new file is already saved
					log.Printf("Failed to delete old document file: %v", err)
				}

				// Create new document record
				doc := models.ResearchDocument{
					ResearchID: uint(id),
					FilePath:   filepath,
					FileType:   fileType,
					FileHash:   newFileHash,
					UploadedBy: user.ID,
				}

				if err := tx.Create(&doc).Error; err != nil {
					tx.Rollback()
					// Clean up the new file
					os.Remove(filepath)
					utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create document record", nil)
					return
				}
			}
		} else {
			// No existing document, create new one
			// Create directory if it doesn't exist
			if err := os.MkdirAll(ResearchDocPath, 0755); err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create upload directory", nil)
				return
			}

			// Generate unique filename
			ext := filepath.Ext(header.Filename)
			filename := fmt.Sprintf("%d_%s%s", id, utils.GenerateUniqueString(), ext)
			filepath := filepath.Join(ResearchDocPath, filename)

			// Reset file pointer before saving
			file.Seek(0, 0)

			// Save the new file
			if err := c.SaveUploadedFile(header, filepath); err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save document", nil)
				return
			}

			// Create new document record
			doc := models.ResearchDocument{
				ResearchID: uint(id),
				FilePath:   filepath,
				FileType:   fileType,
				FileHash:   newFileHash,
				UploadedBy: user.ID,
			}

			if err := tx.Create(&doc).Error; err != nil {
				tx.Rollback()
				// Clean up the new file
				os.Remove(filepath)
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create document record", nil)
				return
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to commit transaction: %v", err), nil)
		return
	}

	// Get updated research with category and user information
	var updatedResearch struct {
		models.Research
		CategoryName string `gorm:"column:category_name"`
		UserName     string `gorm:"column:user_name"`
	}
	if err := config.DB.Model(&models.Research{}).
		Select("researches.*, research_categories.name as category_name, CONCAT(users.first_name, ' ', COALESCE(users.last_name, '')) as user_name").
		Joins("LEFT JOIN research_categories ON researches.category_id = research_categories.id").
		Joins("LEFT JOIN users ON researches.created_by = users.id").
		Where("researches.id = ?", id).
		First(&updatedResearch).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch updated research", nil)
		return
	}

	// Get document information
	var doc models.ResearchDocument
	var docResponse *ResearchDocumentResponse
	if err := config.DB.Where("research_id = ? AND is_deleted = ?", id, false).First(&doc).Error; err == nil {
		docResponse = &ResearchDocumentResponse{
			ID:         doc.ID,
			FilePath:   doc.FilePath,
			FileType:   doc.FileType,
			UploadedBy: doc.UploadedBy,
			UserName:   updatedResearch.UserName,
			CreatedAt:  doc.CreatedAt,
			UpdatedAt:  doc.UpdatedAt,
		}
	}

	response := ResearchResponse{
		ID:           updatedResearch.ID,
		Title:        updatedResearch.Title,
		Description:  updatedResearch.Description,
		CategoryID:   updatedResearch.CategoryID,
		CategoryName: updatedResearch.CategoryName,
		CreatedBy:    updatedResearch.CreatedBy,
		UserName:     updatedResearch.UserName,
		CreatedAt:    updatedResearch.CreatedAt,
		UpdatedAt:    updatedResearch.UpdatedAt,
		IsPublished:  updatedResearch.IsPublished,
		Document:     docResponse,
	}

	utils.JSONResponse(c, http.StatusOK, "Research updated successfully", response)
}

// PublishResearch publishes a research (sets is_published to true)
func PublishResearch(c *gin.Context) {
	// Get user from context using the middleware helper
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can publish researches", nil)
		return
	}

	// Get research ID from URL parameter
	researchID := c.Param("id")
	if researchID == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Research ID is required", nil)
		return
	}

	// Convert researchID to uint
	id, err := strconv.ParseUint(researchID, 10, 32)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid research ID format", nil)
		return
	}

	// Check if research exists and is not deleted
	var research models.Research
	if err := config.DB.Where("id = ? AND is_deleted = ?", id, false).First(&research).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusNotFound, "Research not found", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch research: %v", err), nil)
		}
		return
	}

	// Check if research is already published
	if research.IsPublished {
		utils.JSONResponse(c, http.StatusConflict, "Research is already published", nil)
		return
	}

	// Check if research has a document
	var doc models.ResearchDocument
	if err := config.DB.Where("research_id = ? AND is_deleted = ?", id, false).First(&doc).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusBadRequest, "Cannot publish research without a document", nil)
		} else {
			utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to check research document: %v", err), nil)
		}
		return
	}

	// Verify document file exists
	if _, err := os.Stat(doc.FilePath); os.IsNotExist(err) {
		utils.JSONResponse(c, http.StatusBadRequest, "Cannot publish research: document file not found", nil)
		return
	}

	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Update research to published status
	updates := map[string]interface{}{
		"is_published": true,
		"updated_at":   time.Now(),
	}

	if err := tx.Model(&research).Updates(updates).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to publish research: %v", err), nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to commit transaction: %v", err), nil)
		return
	}

	// Get updated research with category and user information
	var updatedResearch struct {
		models.Research
		CategoryName string `gorm:"column:category_name"`
		UserName     string `gorm:"column:user_name"`
	}
	if err := config.DB.Model(&models.Research{}).
		Select("researches.*, research_categories.name as category_name, CONCAT(users.first_name, ' ', COALESCE(users.last_name, '')) as user_name").
		Joins("LEFT JOIN research_categories ON researches.category_id = research_categories.id").
		Joins("LEFT JOIN users ON researches.created_by = users.id").
		Where("researches.id = ?", id).
		First(&updatedResearch).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch updated research", nil)
		return
	}

	// Prepare document response
	docResponse := ResearchDocumentResponse{
		ID:         doc.ID,
		FilePath:   doc.FilePath,
		FileType:   doc.FileType,
		UploadedBy: doc.UploadedBy,
		UserName:   updatedResearch.UserName,
		CreatedAt:  doc.CreatedAt,
		UpdatedAt:  doc.UpdatedAt,
	}

	response := ResearchResponse{
		ID:           updatedResearch.ID,
		Title:        updatedResearch.Title,
		Description:  updatedResearch.Description,
		CategoryID:   updatedResearch.CategoryID,
		CategoryName: updatedResearch.CategoryName,
		CreatedBy:    updatedResearch.CreatedBy,
		UserName:     updatedResearch.UserName,
		CreatedAt:    updatedResearch.CreatedAt,
		UpdatedAt:    updatedResearch.UpdatedAt,
		IsPublished:  updatedResearch.IsPublished,
		Document:     &docResponse,
	}

	utils.JSONResponse(c, http.StatusOK, "Research published successfully", response)

	// Notify all patients about the published research
	err = services.NotifyUsersByRoles(
		config.DB,
		nil, // no transaction
		[]string{"patient"},
		"New Research Published",
		"Research",
		fmt.Sprintf("A new research '%s' has been published. Check it out!", updatedResearch.Title),
		updatedResearch.UserName,
		"researches",
		updatedResearch.ID,
		map[string]interface{}{
			"research_id": updatedResearch.ID,
			"title":       updatedResearch.Title,
		},
	)
	if err != nil {
		log.Printf("Failed to notify patients about published research: %v", err)
	}
}
