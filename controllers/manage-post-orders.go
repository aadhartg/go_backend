// controllers/manage-post-orders.go

package controllers

import (
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// PatientInfo represents the decrypted patient details
type PatientInfo struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Age       string `json:"age"`
	Gender    string `json:"gender"`
}

// KitRegistrationResponse represents the response for kit registration list
type KitRegistrationResponse struct {
	ID               uint      `json:"id"`
	PatientFirstName string    `json:"patient_first_name"`
	PatientLastName  string    `json:"patient_last_name"`
	PatientEmail     string    `json:"patient_email"`
	PatientAge       string    `json:"patient_age"`
	PatientGender    string    `json:"patient_gender"`
	KitStatus        string    `json:"kit_status"`
	CreatedAt        time.Time `json:"created_at"`
	BarcodeNumber    string    `json:"barcode_number"`
	FilePath         string    `json:"file_path"`
	Reason           string    `json:"reason"`
	// Added customer details
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	// Added order details
	OrderNumber string `json:"order_number"`
}

// AdminUsersResponse represents the structured response for admin users list.
type KitsResponse struct {
	Page         int                       `json:"page"`
	PerPage      int                       `json:"per_page"`
	Sort         string                    `json:"sort"`
	SortColumn   string                    `json:"sort_column"`
	SearchText   string                    `json:"search_text"`
	Status       string                    `json:"status"`
	TotalRecords int64                     `json:"total_records"`
	TotalPages   int                       `json:"total_pages"`
	Records      []KitRegistrationResponse `json:"records"`
}

type UpdateStatusRequest struct {
	Status     string `json:"status" form:"status"`
	Reason     string `json:"reason,omitempty" form:"reason"`
	Type       string `json:"type"`        // new or old
	LabID      uint   `json:"lab_id"`      // required when type is old
	LabName    string `json:"lab_name"`    // required when type is new
	LabAddress string `json:"lab_address"` // required when type is new
	NHINumber  string `json:"nhi_number"`  // required when type is new
}

type FileUploadRequest struct {
	File uint `json:"file" form:"file"`
}

func GetPatientRegisterKitList(c *gin.Context) {

	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "status"}

	// Parse query parameters with default values
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidQueryParameters, nil)
		return
	}

	// Validate and parse pagination parameters
	page, perPage, err := services.ValidatePaginationParams(query)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate and parse sorting parameters
	sortColumn, sort, err := services.ValidateSortParams(query)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate status parameter
	status, err := services.ValidateStatusParam(query)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate search text
	searchText := strings.TrimSpace(query.Get("search_text"))

	// Fetch kit registrations
	totalCount, registrations, err := services.FetchKitRegistrations(page, perPage, sortColumn, sort, searchText, status)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch kit registrations", err)
		return
	}

	// Map registrations to the desired type
	mappedRegistrations := make([]KitRegistrationResponse, len(registrations))
	for i, reg := range registrations {
		mappedRegistrations[i] = KitRegistrationResponse{
			ID:               reg.ID,
			PatientFirstName: reg.PatientFirstName,
			PatientLastName:  reg.PatientLastName,
			PatientAge:       reg.PatientAge,
			PatientGender:    reg.PatientGender,
			PatientEmail:     reg.PatientEmail,
			KitStatus:        reg.KitStatus,
			CreatedAt:        reg.CreatedAt,
			BarcodeNumber:    reg.BarcodeNumber,
			FilePath:         reg.FilePath,
			Reason:           reg.Reason,
			CustomerName:     reg.CustomerName,
			CustomerEmail:    reg.CustomerEmail,
			OrderNumber:      reg.OrderNumber,
		}
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))

	// Prepare response using KitsResponse structure
	response := KitsResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		Status:       status,
		TotalRecords: int64(totalCount),
		TotalPages:   totalPages,
		Records:      mappedRegistrations,
	}

	utils.JSONResponse(c, http.StatusOK, "Kit registrations fetched successfully", response)
}

// UpdatePatientStatus handles the update of a patient's status
func UpdatePatientStatus(c *gin.Context) {
	// Get kit ID from URL parameters
	patientID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidKitID, nil)
		return
	}

	// Check if the patient exists
	exists, err := services.CheckPatientExists(uint(patientID))
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to check patient existence", err)
		return
	}
	if !exists {
		utils.JSONResponse(c, http.StatusNotFound, "Patient not found", nil)
		return
	}

	// Parse and validate request
	var req UpdateStatusRequest
	allowedFields := []string{"status", "reason", "type", "lab_id", "lab_name", "lab_address", "nhi_number"}

	if err := utils.ParseRequestBody(c.Request, &req, allowedFields); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	req.Status = strings.ToLower(req.Status)
	validStatuses := map[string]bool{
		"not-received": true,
		"received":     true,
		"reject":       true,
		"send":         true,
	}
	if !validStatuses[req.Status] {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid status value", nil)
		return
	}

	// Additional validations for 'send' status
	if req.Status == "send" {
		req.Type = strings.TrimSpace(strings.ToLower(req.Type))
		req.LabName = strings.TrimSpace(req.LabName)
		req.LabAddress = strings.TrimSpace(req.LabAddress)
		req.NHINumber = strings.TrimSpace(req.NHINumber)
		if err := services.ValidateSendStatus(req.Type, req.LabName, req.LabAddress, req.NHINumber, req.LabID); err != nil {
			utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
	}

	// Fetch the current status of the patient
	currentStatus, err := services.GetPatientStatus(uint(patientID))
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch patient status", err)
		return
	}

	// Check if the new status is already assigned
	if strings.ToLower(currentStatus) == strings.ToLower(req.Status) {
		utils.JSONResponse(c, http.StatusBadRequest, "The new status is the same as the current status", nil)
		return
	}

	// Validate status transitions
	err = services.ValidateStatusTransition(currentStatus, req.Status, req.Reason)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Process lab information for 'send' status
	var labID uint
	if req.Status == "send" {
		req.Type = strings.TrimSpace(strings.ToLower(req.Type))
		req.LabName = strings.TrimSpace(req.LabName)
		req.LabAddress = strings.TrimSpace(req.LabAddress)
		req.NHINumber = strings.TrimSpace(req.NHINumber)
		labID, err = services.ProcessLabInformation(req.Type, req.LabName, req.LabAddress, req.NHINumber, req.LabID)
		if err != nil {
			utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
	}

	// Update the status in the database
	err = services.UpdatePatientStatus(uint(patientID), req.Status, req.Reason, labID)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update patient status", err)
		return
	}

	utils.JSONResponse(c, http.StatusOK, "Patient status updated successfully", nil)
}

// FileUpload handles file uploads for a patient
func FileUpload(c *gin.Context) {
	// Extract user ID from URL
	patientID := c.Param("id")

	// Parse multipart form
	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB max file size
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "File too large or invalid form", nil)
		return
	}

	// Get file from request
	file, handler, err := c.Request.FormFile("file")
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "No file uploaded", nil)
		return
	}
	defer file.Close()

	// Validate file type
	fileExt := strings.ToLower(filepath.Ext(handler.Filename))
	if fileExt != ".pdf" && fileExt != ".xml" {
		utils.JSONResponse(c, http.StatusBadRequest, "Only PDF and XML files are allowed", nil)
		return
	}

	// Check file size
	if handler.Size > (10 << 20) { // 10 MB
		utils.JSONResponse(c, http.StatusBadRequest, "File size exceeds 10 MB limit", nil)
		return
	}

	// Call service to handle file upload
	err = services.HandleFileUpload(patientID, file, handler.Filename, fileExt)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// Successful response
	utils.JSONResponse(c, http.StatusOK, "File uploaded successfully", nil)
}
