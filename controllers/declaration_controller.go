package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"

	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
)

// CreateDeclarationRequest represents the request body for creating a declaration
type CreateDeclarationRequest struct {
	Statement    string              `json:"statement" binding:"required"`
	Title        string              `json:"title" binding:"required,min=3,max=255"`
	ResponseType models.ResponseType `json:"response_type" binding:"required,oneof=yes_no text multi_text"`
}

// UpdateDeclarationRequest represents the request body for updating a declaration
type UpdateDeclarationRequest struct {
	Statement    string              `json:"statement" binding:"omitempty,min=1"`
	Title        string              `json:"title" binding:"omitempty,min=3,max=255"`
	ResponseType models.ResponseType `json:"response_type" binding:"omitempty,oneof=yes_no text multi_text"`
}

// DeclarationResponse represents a single declaration in the response
type DeclarationResponse struct {
	ID            uint      `json:"id"`
	Statement     string    `json:"statement"`
	Title         string    `json:"title"`
	ResponseType  string    `json:"response_type"`
	CreatedBy     uint      `json:"created_by"`
	UpdatedBy     *uint     `json:"updated_by"`
	UserName      string    `json:"user_name"`
	UpdatedByName string    `json:"updated_by_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DeclarationsListResponse represents the structured response for declarations list
type DeclarationsListResponse struct {
	Page         int                   `json:"page"`
	PerPage      int                   `json:"per_page"`
	Sort         string                `json:"sort"`
	SortColumn   string                `json:"sort_column"`
	SearchText   string                `json:"search_text"`
	TotalRecords int64                 `json:"total_records"`
	TotalPages   int                   `json:"total_pages"`
	Records      []DeclarationResponse `json:"records"`
}

// BatchCreateDeclarationsRequest now includes form_name and form_description
// Each declaration must have statement, title, response_type, and (optionally) options
//
type BatchCreateDeclarationsRequest struct {
	FormName        string `json:"form_name" binding:"required,min=3,max=255"`
	FormDescription string `json:"form_description"`
	Declarations []struct {
		Statement    string              `json:"statement" binding:"required"`
		Title        string              `json:"title" binding:"required,min=3,max=255"`
		ResponseType models.ResponseType `json:"response_type" binding:"required,oneof=yes_no text multi_text"`
		Options      models.StringArray   `json:"options"`
	} `json:"declarations" binding:"required,dive,required"`
}

// BatchUpdateDeclarationsRequest represents the request body for batch editing
// Each item must have id and at least one updatable field
//
type BatchUpdateDeclarationsRequest struct {
	FormID          uint   `json:"form_id" binding:"required"`
	FormName        string `json:"form_name"`
	FormDescription string `json:"form_description"`
	Declarations []struct {
		ID          uint                `json:"id"`
		Statement   *string             `json:"statement"`
		Title       *string             `json:"title"`
		ResponseType *models.ResponseType `json:"response_type"`
		Options     *models.StringArray  `json:"options"`
	} `json:"declarations" binding:"required,dive,required"`
}

// checkDeclarationTitleExists checks if a declaration with the given title already exists
func checkDeclarationTitleExists(title string, excludeID uint) (bool, error) {
	var count int64
	query := config.DB.Model(&models.Declaration{}).Where("title = ? AND is_deleted = ?", title, false)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// BatchCreateDeclarations creates multiple declarations in one request
func CreateDeclaration(c *gin.Context) {
	var req BatchCreateDeclarationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can upload videos", nil)
		return
	}

	if len(req.Declarations) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "At least one declaration is required"})
		return
	}

	tx := config.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to start transaction"})
		return
	}

	// Create the form
	form := models.DeclarationForm{
		Name:        req.FormName,
		Description: req.FormDescription,
		CreatedBy:   user.ID,
		CreatedAt:   time.Now(),
	}
	if err := tx.Create(&form).Error; err != nil {
		tx.Rollback()
		// Check for unique constraint violation (Postgres and MySQL)
		var myErr *mysql.MySQLError
		if errors.As(err, &myErr) && myErr.Number == 1062 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "A declaration form with this name already exists. Please choose a different name."})
			return
		}
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") || strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "A declaration form with this name already exists. Please choose a different name."})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create declaration form: " + err.Error()})
		return
	}

	created := make([]DeclarationResponse, 0, len(req.Declarations))
	titles := map[string]bool{}
	for _, d := range req.Declarations {
		// Title uniqueness in request (per form)
		if titles[d.Title] {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Duplicate title in request: " + d.Title})
			return
		}
		titles[d.Title] = true

		// Use service for creation and validation
		declaration, err := services.CreateDeclaration(tx, services.CreateDeclarationInput{
			FormID:       form.ID,
			Statement:    d.Statement,
			Title:        d.Title,
			ResponseType: d.ResponseType,
			Options:      d.Options,
			CreatedBy:    user.ID,
		})
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		// Get the user's name for the response
		var userName string
		tx.Model(&models.User{}).
			Select("CONCAT(first_name, ' ', COALESCE(last_name, ''))").
			Where("id = ?", user.ID).
			Scan(&userName)

		created = append(created, DeclarationResponse{
			ID:           declaration.ID,
			Statement:    declaration.Statement,
			Title:        declaration.Title,
			ResponseType: string(declaration.ResponseType),
			CreatedBy:    declaration.CreatedBy,
			UserName:     userName,
			CreatedAt:    declaration.CreatedAt,
			UpdatedAt:    declaration.UpdatedAt,
		})
	}

	tx.Commit()
	c.JSON(http.StatusCreated, gin.H{
		"message": "Declarations form created successfully",
		"form":    form,
		"declarations": created,
	})
}

// ListDeclarations retrieves a list of declarations with pagination and filtering
func ListDeclarations(c *gin.Context) {
	// Check authorization
	if _, ok := middlewares.GetUserFromContext(c); !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "form_id"}

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
	validSortColumns := []string{"label", "created_at"}
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
	}

	// Optional 'form_id'
	formID := uint(0)
	if val := query.Get("form_id"); val != "" {
		if id, err := strconv.ParseUint(val, 10, 64); err == nil {
			formID = uint(id)
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid form_id parameter", nil)
			return
		}
	}

	// Parse search text for date patterns using the utility function
	dateResult := utils.ParseSearchTextForDate(searchText)
	
	// Convert search text to lowercase for case-insensitive search
	searchPattern := strings.ToLower(searchText)
	
	// First, get the total count of unique forms that have declarations
	var totalRecords int64
	formCountQuery := config.DB.Model(&models.Declaration{}).
		Distinct("form_id").
		Where("is_deleted = ?", false)
	
	if formID > 0 {
		formCountQuery = formCountQuery.Where("form_id = ?", formID)
	}
	
	if searchText != "" {
		formCountQuery = formCountQuery.Joins("JOIN declaration_forms df ON declarations.form_id = df.id")
		
		// Use the utility function to build the WHERE clause with correct column names
		// Map: title -> name, statement -> description, name -> name
		whereClause, args := utils.BuildDateSearchQueryWithColumns(formCountQuery, searchPattern, dateResult, "df", "name", "description", "name")
		formCountQuery = formCountQuery.Where(whereClause, args...)
	}
	
	if err := formCountQuery.Count(&totalRecords).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Calculate total pages
	var totalPages int
	if totalRecords == 0 {
		totalPages = 0
	} else {
		totalPages = int((totalRecords + int64(perPage) - 1) / int64(perPage))
	}

	// Build query to get forms with their declaration counts, sorted by form-level fields
	formQuery := config.DB.Model(&models.DeclarationForm{}).
		Select("declaration_forms.*, COUNT(declarations.id) as declaration_count").
		Joins("JOIN declarations ON declaration_forms.id = declarations.form_id").
		Where("declarations.is_deleted = ?", false).
		Group("declaration_forms.id")

	if formID > 0 {
		formQuery = formQuery.Where("declaration_forms.id = ?", formID)
	}

	// Apply search filter if provided
	if searchText != "" {
		// Use the utility function to build the WHERE clause with correct column names
		// Map: title -> name, statement -> description, name -> name
		whereClause, args := utils.BuildDateSearchQueryWithColumns(formQuery, searchPattern, dateResult, "declaration_forms", "name", "description", "name")
		formQuery = formQuery.Where(whereClause, args...)
	
	}

	// Apply sorting based on form-level fields
	if sortColumn == "label" {
		formQuery = formQuery.Order("declaration_forms.name " + sort)
	} else if sortColumn == "created_at" {
		formQuery = formQuery.Order("declaration_forms.created_at " + sort)
	}

	// Apply pagination
	offset := (page - 1) * perPage
	var formsWithCounts []struct {
		models.DeclarationForm
		DeclarationCount int64 `gorm:"column:declaration_count"`
	}
	if err := formQuery.Limit(perPage).Offset(offset).Find(&formsWithCounts).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Extract form IDs for fetching declarations
	formIDs := make([]uint, 0, len(formsWithCounts))
	for _, f := range formsWithCounts {
		formIDs = append(formIDs, f.ID)
	}

	// Fetch all declarations for the paginated forms
	var declarations []struct {
		models.Declaration
		UserName      string `gorm:"column:user_name"`
		UpdatedByName string `gorm:"column:updated_by_name"`
	}
	if len(formIDs) > 0 {
		declarationQuery := config.DB.Model(&models.Declaration{}).
			Select("declarations.*, "+
				"CONCAT(creator.first_name, ' ', COALESCE(creator.last_name, '')) as user_name, "+
				"CONCAT(updater.first_name, ' ', COALESCE(updater.last_name, '')) as updated_by_name").
			Joins("LEFT JOIN users creator ON declarations.created_by = creator.id").
			Joins("LEFT JOIN users updater ON declarations.updated_by = updater.id").
			Where("declarations.is_deleted = ? AND declarations.form_id IN (?)", false, formIDs)
		
		if err := declarationQuery.Find(&declarations).Error; err != nil {
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
			return
		}
	}

	// Group declarations by form and maintain the order from formsWithCounts
	formMap := map[uint]*struct {
		Form        *models.DeclarationForm   `json:"form"`
		Declarations []interface{}   `json:"declarations"`
	}{}
	
	// Initialize formMap with the forms in the correct order
	for _, f := range formsWithCounts {
		formMap[f.ID] = &struct {
			Form        *models.DeclarationForm   `json:"form"`
			Declarations []interface{}   `json:"declarations"`
		}{Form: &f.DeclarationForm}
	}
	
	// Group declarations by form
	for _, d := range declarations {
		if formGroup, ok := formMap[d.FormID]; ok {
			resp := DeclarationResponse{
				ID:            d.ID,
				Statement:     d.Statement,
				Title:         d.Title,
				ResponseType:  string(d.ResponseType),
				CreatedBy:     d.CreatedBy,
				UpdatedBy:     d.UpdatedBy,
				UserName:      d.UserName,
				UpdatedByName: d.UpdatedByName,
				CreatedAt:     d.CreatedAt,
				UpdatedAt:     d.UpdatedAt,
			}
			// Add options if response_type is text
			if d.ResponseType == models.ResponseTypeText {
				// Use a map[string]interface{} for the declaration to include options
				declWithOptions := map[string]interface{}{
					"id":            d.ID,
					"statement":     d.Statement,
					"title":         d.Title,
					"response_type": string(d.ResponseType),
					"created_by":    d.CreatedBy,
					"updated_by":    d.UpdatedBy,
					"user_name":     d.UserName,
					"updated_by_name": d.UpdatedByName,
					"created_at":    d.CreatedAt,
					"updated_at":    d.UpdatedAt,
					"options":       d.Options,
				}
				formGroup.Declarations = append(formGroup.Declarations, declWithOptions)
			} else {
				formGroup.Declarations = append(formGroup.Declarations, resp)
			}
		}
	}

	// Prepare grouped response in the requested flat format, maintaining the order from formsWithCounts
	grouped := make([]interface{}, 0, len(formsWithCounts))
	for _, f := range formsWithCounts {
		if formGroup, ok := formMap[f.ID]; ok {
			grouped = append(grouped, map[string]interface{}{
				"id":          formGroup.Form.ID,
				"label":       formGroup.Form.Name,
				"description": formGroup.Form.Description,
				"created_by":  formGroup.Form.CreatedBy,
				"created_at":  formGroup.Form.CreatedAt,
				"declarations": formGroup.Declarations,
			})
		}
	}

	response := map[string]interface{}{
		"page":         page,
		"per_page":     perPage,
		"sort":         sort,
		"sort_column":  sortColumn,
		"search_text":  searchText,
		"total_records": totalRecords,
		"total_pages":  totalPages,
		"groups":       grouped,
	}

	utils.JSONResponse(c, http.StatusOK, "Declarations fetched successfully", response)
}

// BatchUpdateDeclarations updates multiple declarations in one request
func UpdateDeclaration(c *gin.Context) {
	var req BatchUpdateDeclarationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can upload videos", nil)
		return
	}

	if len(req.Declarations) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "At least one declaration is required"})
		return
	}

	tx := config.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to start transaction"})
		return
	}

	// Fetch and update the form if needed
	var form models.DeclarationForm
	if err := tx.First(&form, req.FormID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Declaration form not found"})
		return
	}
	formUpdated := false
	if req.FormName != "" && req.FormName != form.Name {
		form.Name = req.FormName
		formUpdated = true
	}
	if req.FormDescription != "" && req.FormDescription != form.Description {
		form.Description = req.FormDescription
		formUpdated = true
	}
	if formUpdated {
		if err := tx.Save(&form).Error; err != nil {
			tx.Rollback()
			// Improved error message for unique constraint violation
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") || strings.Contains(err.Error(), "Duplicate entry") {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "A declaration form with this name already exists. Please choose a different name."})
				return
			}
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update declaration form: " + err.Error()})
			return
		}
	}

	updated := make([]DeclarationResponse, 0, len(req.Declarations))
	titles := map[string]uint{} // title -> id
	requestIDs := make([]uint, 0) // <-- Move requestIDs here so we can append new IDs as we go
	for _, d := range req.Declarations {
		if d.ID == 0 {
			// Create new declaration using service
			if d.Title == nil || *d.Title == "" {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Title is required for new declaration"})
				return
			}
			if otherID, exists := titles[*d.Title]; exists && otherID == 0 {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Duplicate title in request: " + *d.Title})
				return
			}
			titles[*d.Title] = 0
			if d.ResponseType == nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Response type is required for new declaration: " + *d.Title})
				return
			}
			statement := ""
			if d.Statement != nil {
				statement = *d.Statement
			}
			options := models.StringArray{}
			if d.Options != nil {
				options = *d.Options
			}
			declaration, err := services.CreateDeclaration(tx, services.CreateDeclarationInput{
				FormID:       req.FormID,
				Statement:    statement,
				Title:        *d.Title,
				ResponseType: *d.ResponseType,
				Options:      options,
				CreatedBy:    user.ID,
			})
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
				return
			}
			// Get the user's name for the response
			var userName string
			tx.Model(&models.User{}).
				Select("CONCAT(first_name, ' ', COALESCE(last_name, ''))").
				Where("id = ?", user.ID).
				Scan(&userName)
			updated = append(updated, DeclarationResponse{
				ID:           declaration.ID,
				Statement:    declaration.Statement,
				Title:        declaration.Title,
				ResponseType: string(declaration.ResponseType),
				CreatedBy:    declaration.CreatedBy,
				UpdatedBy:    nil,
				UserName:     userName,
				UpdatedByName: "",
				CreatedAt:    declaration.CreatedAt,
				UpdatedAt:    declaration.UpdatedAt,
			})
			requestIDs = append(requestIDs, declaration.ID) // <-- Add new declaration ID to requestIDs
			continue
		}
		// Fetch declaration
		var declaration models.Declaration
		if err := tx.Unscoped().First(&declaration, d.ID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Declaration not found: id=" + strconv.Itoa(int(d.ID))})
			return
		}
		if declaration.IsDeleted || declaration.DeletedAt.Valid {
			tx.Rollback()
			c.JSON(http.StatusGone, ErrorResponse{Error: "Declaration does not exist: id=" + strconv.Itoa(int(d.ID))})
			return
		}
		// Ensure declaration belongs to the form
		if declaration.FormID != req.FormID {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Declaration does not belong to the specified form: id=" + strconv.Itoa(int(d.ID))})
			return
		}

		// Prepare updates
		updates := make(map[string]interface{})
		if d.Statement != nil {
			updates["statement"] = *d.Statement
		}
		if d.Title != nil && *d.Title != declaration.Title {
			// Check for duplicate in request
			if otherID, exists := titles[*d.Title]; exists && otherID != d.ID {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Duplicate title in request: " + *d.Title})
				return
			}
			titles[*d.Title] = d.ID
			// Check for duplicate in DB (within the form)
			var count int64
			tx.Model(&models.Declaration{}).Where("form_id = ? AND title = ? AND id != ?", req.FormID, *d.Title, d.ID).Count(&count)
			if count > 0 {
				tx.Rollback()
				c.JSON(http.StatusConflict, ErrorResponse{Error: "The title '" + *d.Title + "' already exists. Please enter a unique declaration title.",})
				return
			}
			updates["title"] = *d.Title
		}
		if d.ResponseType != nil {
			updates["response_type"] = *d.ResponseType
		}
		if d.Options != nil {
			updates["options"] = *d.Options
		}
		// Validation for options
		finalType := declaration.ResponseType
		if d.ResponseType != nil {
			finalType = *d.ResponseType
		}
		finalOptions := declaration.Options
		if d.Options != nil {
			finalOptions = *d.Options
		}
		if finalType == models.ResponseTypeText {
			if len(finalOptions) == 0 {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Options are required for text response_type: id=" + strconv.Itoa(int(d.ID))})
				return
			}
		} else {
			// If type is not text, always clear options
			finalOptions = models.StringArray{}
			updates["options"] = finalOptions
		}
		// Always update the UpdatedBy field
		userID := user.ID
		updates["updated_by"] = &userID

		if err := tx.Model(&declaration).Updates(updates).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update declaration: id=" + strconv.Itoa(int(d.ID))})
			return
		}

		// Get the user's name for the response
		var userName string
		tx.Model(&models.User{}).
			Select("CONCAT(first_name, ' ', COALESCE(last_name, ''))").
			Where("id = ?", declaration.CreatedBy).
			Scan(&userName)
		var updatedByName string
		tx.Model(&models.User{}).
			Select("CONCAT(first_name, ' ', COALESCE(last_name, ''))").
			Where("id = ?", user.ID).
			Scan(&updatedByName)

		updated = append(updated, DeclarationResponse{
			ID:            declaration.ID,
			Statement:     func() string { if v, ok := updates["statement"].(string); ok { return v }; return declaration.Statement }(),
			Title:         func() string { if v, ok := updates["title"].(string); ok { return v }; return declaration.Title }(),
			ResponseType:  string(finalType),
			CreatedBy:     declaration.CreatedBy,
			UpdatedBy:     &userID,
			UserName:      userName,
			UpdatedByName: updatedByName,
			CreatedAt:     declaration.CreatedAt,
			UpdatedAt:     time.Now(),
		})
		requestIDs = append(requestIDs, d.ID) // <-- Add existing declaration ID to requestIDs
	}
	// After processing all updates/adds, hard-delete declarations not present in the request
	var toDelete []models.Declaration
	if len(requestIDs) > 0 {
		tx.Where("form_id = ? AND id NOT IN (?)", req.FormID, requestIDs).Find(&toDelete)
	} else {
		tx.Where("form_id = ?", req.FormID).Find(&toDelete)
	}
	for _, decl := range toDelete {
		// First, delete all related declaration responses
		if err := tx.Unscoped().Where("declaration_id = ?", decl.ID).Delete(&models.DeclarationResponse{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete related declaration responses: " + err.Error()})
			return
		}
		
		// Then, delete all related consent declarations
		if err := tx.Unscoped().Where("declaration_id = ?", decl.ID).Delete(&models.ConsentDeclaration{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete related consent declarations: " + err.Error()})
			return
		}
		
		// Finally, delete the declaration itself
		if err := tx.Unscoped().Delete(&decl).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete declaration: " + err.Error()})
			return
		}
	}
	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message": "Form and declarations updated successfully",
		"form": map[string]interface{}{
			"id":          form.ID,
			"label":       form.Name,
			"description": form.Description,
			"created_by":  form.CreatedBy,
			"created_at":  form.CreatedAt,
		},
		"declarations": updated,
	})
}

// DeleteDeclarationForm hard deletes a declaration form and all associated declarations
func DeleteDeclarationForm(c *gin.Context) {
	// Check authorization
	if _, ok := middlewares.GetUserFromContext(c); !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	formID := c.Param("form_id")
	if formID == "" {
		formID = c.Param("id")
	}
	if formID == "" {
		formID = c.Query("form_id")
	}
	if formID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "form_id is required"})
		return
	}
	id, err := strconv.ParseUint(formID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid form_id"})
		return
	}

	tx := config.DB.Begin()
	
	// First, get all declarations for this form
	var declarations []models.Declaration
	if err := tx.Where("form_id = ?", id).Find(&declarations).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to fetch declarations for form"})
		return
	}
	
	// Delete all related records for each declaration
	for _, decl := range declarations {
		// Delete all related declaration responses
		if err := tx.Unscoped().Where("declaration_id = ?", decl.ID).Delete(&models.DeclarationResponse{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete related declaration responses"})
			return
		}
		
		// Delete all related consent declarations
		if err := tx.Unscoped().Where("declaration_id = ?", decl.ID).Delete(&models.ConsentDeclaration{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete related consent declarations"})
			return
		}
	}
	
	// Hard delete all declarations for this form
	if err := tx.Unscoped().Where("form_id = ?", id).Delete(&models.Declaration{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to hard delete declarations for form"})
		return
	}
	
	// Hard delete the form
	if err := tx.Unscoped().Where("id = ?", id).Delete(&models.DeclarationForm{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "This declaration form cannot be deleted because it is associated with existing E-Consent forms."})
		return
	}
	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message": "Form and all its declarations hard deleted successfully",
		"form_id": id,
	})
}
