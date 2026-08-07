package controllers

import (
	"net/http"
	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// DashboardStats represents the statistics for the admin dashboard
type DashboardStats struct {
	TotalPatients               int64 `json:"total_patients"`
	TotalResearch               int64 `json:"total_research"`
	TotalSignedConsent          int64 `json:"total_signed_consent"`
	ConsentsAssignedNotApproved int64 `json:"consents_assigned_not_approved"`
	ConsentsExpiringSoon        int64 `json:"consents_expiring_soon"`
	ConsentsWithdrawnOrReviewed int64 `json:"consents_withdrawn_or_reviewed"`
	TotalGenomicReports         int64 `json:"total_genomic_reports"`
	GenomicReportsNotPublished  int64 `json:"genomic_reports_not_published"`
}

// DropdownData represents the data for dropdowns in e-consent form creation
type DropdownData struct {
	Videos       []VideoDropdown       `json:"videos"`
	Research     []ResearchDropdown    `json:"research"`
	Declarations []DeclarationDropdown `json:"declarations"`
	Patients     []UserDropdown        `json:"patients"`
	Coordinators []UserDropdown        `json:"coordinators"`
}

// VideoDropdown represents video data for dropdown
type VideoDropdown struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Description string `json:"description"`
}

// ResearchDropdown represents research data for dropdown
type ResearchDropdown struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

// DeclarationDropdown represents declaration data for dropdown
type DeclarationDropdown struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

// UserDropdown represents user data for dropdown
type UserDropdown struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// GetDashboardStats returns statistics for the admin dashboard
func GetDashboardStats(c *gin.Context) {
	// Get user from context (set by auth middleware)
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Unauthorized",
		})
		return
	}

	// Type assert user to models.User
	userModel, ok := user.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Invalid user data",
		})
		return
	}

	// Check if user has admin or super-admin role
	if userModel.RoleID != 1 && userModel.RoleID != 2 {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Access denied. Admin privileges required",
		})
		return
	}

	var stats DashboardStats

	// Count total patients (users with patient role)
	if err := config.DB.Model(&models.User{}).
		Joins("JOIN roles ON users.role_id = roles.id").
		Where("roles.name = ? AND users.is_deleted = ?", "patient", false).
		Count(&stats.TotalPatients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to count patients",
		})
		return
	}

	// Count total research documents
	if err := config.DB.Model(&models.ResearchDocument{}).
		Where("is_deleted = ?", false).
		Count(&stats.TotalResearch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to count research documents",
		})
		return
	}

	// Count total signed consents (status = approved)
	if err := config.DB.Model(&models.EConsentAssignment{}).
		Where("status = ? AND is_deleted = ?", "approved", false).
		Count(&stats.TotalSignedConsent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to count signed consents",
		})
		return
	}

	// // Count consents which are assigned but not approved (excluding withdrawn or reviewed)
	// if err := config.DB.Model(&models.EConsentAssignment{}).
	// 	Where("status IN (?, ?) AND is_deleted = ?", "assigned", "in_progress", false).
	// 	Count(&stats.ConsentsAssignedNotApproved).Error; err != nil {
	// 	c.JSON(http.StatusInternalServerError, ErrorResponse{
	// 		Error: "Failed to count assigned but not approved consents",
	// 	})
	// 	return
	// }

	// Count consents which are assigned but not approved (excluding withdrawn or reviewed)
	if err := config.DB.Model(&models.EConsentAssignment{}).
		Where("status = ? AND is_deleted = ?", "assigned", false).
		Count(&stats.ConsentsAssignedNotApproved).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to count assigned consents",
		})
		return
	}

	//	
	twoDaysFromNow := time.Now().AddDate(0, 0, 2)
	
	// Count assignments that are in "assigned" status and will expire in 2 days
	if err := config.DB.Model(&models.EConsentAssignment{}).
		Joins("JOIN e_consent_forms ON e_consent_assignments.e_consent_form_id = e_consent_forms.id").
		Where("e_consent_assignments.status = ? AND e_consent_assignments.is_deleted = ? AND e_consent_forms.is_deleted = ? AND e_consent_forms.expiry_time <= ? AND e_consent_forms.expiry_time > ?",
			"assigned", false, false, twoDaysFromNow, time.Now()).
		Count(&stats.ConsentsExpiringSoon).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to count consents expiring soon",
		})
		return
	}

	// Count consents reviewed
	if err := config.DB.Model(&models.EConsentAssignment{}).
		Where("status = ? AND is_deleted = ?", "reviewed", false).
		Count(&stats.ConsentsWithdrawnOrReviewed).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to count reviewed consents",
		})
		return
	}

	// Count total genomic reports
	if err := config.DB.Model(&models.Genomics{}).
		Where("is_deleted = ?", false).
		Count(&stats.TotalGenomicReports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to count total genomic reports",
		})
		return
	}

	// Count genomic reports which are saved but not published
	if err := config.DB.Model(&models.Genomics{}).
		Where("is_published = ? AND is_deleted = ?", false, false).
		Count(&stats.GenomicReportsNotPublished).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to count unpublished genomic reports",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Dashboard statistics fetched successfully",
		"data":    stats,
	})
}

// GetEConsentDropdownData returns data for e-consent form dropdowns
func GetEConsentDropdownData(c *gin.Context) {
	// Get user from context (set by auth middleware)
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Unauthorized",
		})
		return
	}

	// Type assert user to models.User
	userModel, ok := user.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Invalid user data",
		})
		return
	}

	// Check if user has admin or super-admin role
	if userModel.RoleID != 1 && userModel.RoleID != 2 {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Access denied. Admin privileges required",
		})
		return
	}

	var dropdownData DropdownData

	// Get videos (id, title)
	if err := config.DB.Model(&models.Video{}).
		Select("id, title, description").
		Where("is_deleted = ?", false).
		Order("title ASC").
		Find(&dropdownData.Videos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to fetch videos",
		})
		return
	}

	// Get research (id, title)
	if err := config.DB.Model(&models.Research{}).
		Select("id, title").
		Where("is_deleted = ? AND is_published = ?", false, true).
		Order("title ASC").
		Find(&dropdownData.Research).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to fetch research",
		})
		return
	}

	// Get declarations (id, title)
	// if err := config.DB.Model(&models.Declaration{}).
	// 	Select("id, title").
	// 	Where("is_deleted = ?", false).
	// 	Order("title ASC").
	// 	Find(&dropdownData.Declarations).Error; err != nil {
	// 	c.JSON(http.StatusInternalServerError, ErrorResponse{
	// 		Error: "Failed to fetch declarations",
	// 	})
	// 	return
	// }

	// Get declaration forms and their respective declarations
	var forms []models.DeclarationForm
	if err := config.DB.Preload("Declarations", "is_deleted = ?", false).Find(&forms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to fetch declaration forms",
		})
		return
	}
	// Prepare the response structure for forms and their declarations
	declarationForms := make([]map[string]interface{}, 0, len(forms))
	for _, form := range forms {
		declarations := make([]map[string]interface{}, 0, len(form.Declarations))
		for _, decl := range form.Declarations {
			declarations = append(declarations, map[string]interface{}{
				"id":            decl.ID,
				"title":         decl.Title,
				"statement":     decl.Statement,
				"response_type": decl.ResponseType,
				"options":       decl.Options,
			})
		}
		declarationForms = append(declarationForms, map[string]interface{}{
			"id":           form.ID,
			"name":         form.Name,
			"description":  form.Description,
			"declarations": declarations,
		})
	}

	// Get patients (id, first_name, last_name) - users with role_id = 4
	var patients []struct {
		ID        uint   `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	if err := config.DB.Model(&models.User{}).
		Select("id, first_name, last_name").
		Where("role_id = ? AND is_deleted = ? AND active_status = ?", 4, false, true).
		Order("first_name ASC, last_name ASC").
		Find(&patients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to fetch patients",
		})
		return
	}

	// Convert patients to UserDropdown format with full name
	dropdownData.Patients = make([]UserDropdown, len(patients))
	for i, patient := range patients {
		dropdownData.Patients[i] = UserDropdown{
			ID:        patient.ID,
			FirstName: patient.FirstName,
			LastName:  patient.LastName,
		}
	}

	// Get coordinators (id, first_name, last_name) - users with role_id = 1 or 2
	var coordinators []struct {
		ID        uint   `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	if err := config.DB.Model(&models.User{}).
		Select("id, first_name, last_name").
		//Where("role_id IN (?, ?) AND is_deleted = ?", 1, 2, false).
		Where("role_id IN (?) AND is_deleted = ?", 5, false).
		Order("first_name ASC, last_name ASC").
		Find(&coordinators).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to fetch coordinators",
		})
		return
	}

	// Convert coordinators to UserDropdown format with full name
	dropdownData.Coordinators = make([]UserDropdown, len(coordinators))
	for i, coordinator := range coordinators {
		dropdownData.Coordinators[i] = UserDropdown{
			ID:        coordinator.ID,
			FirstName: coordinator.FirstName,
			LastName:  coordinator.LastName,
		}
	}

	dropdownDataMap := gin.H{
		"videos":             dropdownData.Videos,
		"research":           dropdownData.Research,
		"patients":           dropdownData.Patients,
		"coordinators":       dropdownData.Coordinators,
		"declarations_forms": declarationForms,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Dropdown data fetched successfully",
		"data":    dropdownDataMap,
	})
}
