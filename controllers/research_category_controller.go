package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

// CreateResearchCategoryRequest represents the request body for creating a research category
type CreateResearchCategoryRequest struct {
	Name string `json:"name" binding:"required,min=3,max=40"`
}

// ResearchCategoryResponse represents a single research category in the response
type ResearchCategoryResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	CreatedBy uint      `json:"created_by"`
	UserName  string    `json:"user_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ResearchCategoriesListResponse represents the structured response for research categories list
type ResearchCategoriesListResponse struct {
	Page         int                        `json:"page"`
	PerPage      int                        `json:"per_page"`
	Sort         string                     `json:"sort"`
	SortColumn   string                     `json:"sort_column"`
	SearchText   string                     `json:"search_text"`
	TotalRecords int64                      `json:"total_records"`
	TotalPages   int                        `json:"total_pages"`
	Records      []ResearchCategoryResponse `json:"records"`
}

// CategoryListResponse represents the simplified response for category list
type CategoryListResponse struct {
	Records []CategoryBasicResponse `json:"records"`
}

// CategoryBasicResponse represents the basic category information
type CategoryBasicResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// checkCategoryNameExists checks if a category with the given name already exists (case-insensitive)
func checkCategoryNameExists(name string, excludeID uint) (bool, error) {
	var count int64
	query := config.DB.Model(&models.ResearchCategory{}).Where("LOWER(name) = LOWER(?) AND is_deleted = ?", name, false)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateResearchCategory creates a new research category
func CreateResearchCategory(c *gin.Context) {
	var req CreateResearchCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get user from context using the middleware helper
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Trim whitespace and validate name
	trimmedName := strings.TrimSpace(req.Name)
	if trimmedName == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Category name cannot be empty"})
		return
	}

	// Check if name already exists (case-insensitive)
	exists, err := checkCategoryNameExists(trimmedName, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to validate category name"})
		return
	}
	if exists {
		// Return a proper error response
		c.JSON(http.StatusConflict, gin.H{
			"status":  http.StatusConflict,
			"message": fmt.Sprintf("A category with the name '%s' already exists", trimmedName),
			"error":   "DUPLICATE_CATEGORY_NAME",
		})
		return
	}

	category := models.ResearchCategory{
		Name:      trimmedName,
		CreatedBy: user.ID,
	}

	if err := config.DB.Create(&category).Error; err != nil {
		// Check for duplicate key error (additional safety check)
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(http.StatusConflict, gin.H{
				"status":  http.StatusConflict,
				"message": fmt.Sprintf("A category with the name '%s' already exists", trimmedName),
				"error":   "DUPLICATE_CATEGORY_NAME",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create research category"})
		return
	}

	// Return the created category
	utils.JSONResponse(c, http.StatusCreated, "Research category created successfully", gin.H{
		"id":         category.ID,
		"name":       category.Name,
		"created_by": category.CreatedBy,
		"created_at": category.CreatedAt,
		"updated_at": category.UpdatedAt,
	})
}

// ListCategories retrieves a list of all research categories
func ListCategories(c *gin.Context) {
	// Check authorization
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can view categories", nil)
		return
	}

	// Get all non-deleted categories
	var categories []models.ResearchCategory
	if err := config.DB.Where("is_deleted = ?", false).Order("name asc").Find(&categories).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Convert to basic response
	records := make([]CategoryBasicResponse, len(categories))
	for i, cat := range categories {
		records[i] = CategoryBasicResponse{
			ID:   cat.ID,
			Name: cat.Name,
		}
	}

	response := CategoryListResponse{
		Records: records,
	}

	utils.JSONResponse(c, http.StatusOK, "Categories fetched successfully", response)
}
