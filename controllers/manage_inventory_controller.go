// manage_inventory_controller
package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"theransticslabs/m/config"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// KitRequest represents the expected request body structure
type KitRequest struct {
	Type                  string      `json:"type" form:"type"`
	SupplierName          string      `json:"supplier_name" form:"supplier_name"`
	SupplierContactNumber string      `json:"supplier_contact_number" form:"supplier_contact_number"`
	SupplierAddress       string      `json:"supplier_address" form:"supplier_address"`
	Quantity              interface{} `json:"quantity" form:"quantity"`
}

type KitsListResponse struct {
	Page         int         `json:"page"`
	PerPage      int         `json:"per_page"`
	Sort         string      `json:"sort"`
	SortColumn   string      `json:"sort_column"`
	SearchText   string      `json:"search_text"`
	TotalRecords int64       `json:"total_records"`
	TotalPages   int         `json:"total_pages"`
	Type         string      `json:"type"`
	Records      []KitDetail `json:"records"`
}

type KitDetail struct {
	ID                    uint            `json:"id"`
	Type                  string          `json:"type"`
	Quantity              int             `json:"quantity"`
	SupplierName          string          `json:"supplier_name"`
	SupplierAddress       string          `json:"supplier_address"`
	SupplierContactNumber string          `json:"supplier_contact_number"`
	Status                bool            `json:"status"`
	CreatedAt             time.Time       `json:"created_at"`
	CreatedBy             KitsUserProfile `json:"created_by"`
}

type KitsUserProfile struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// KitUpdateRequest represents the PATCH request structure
type KitUpdateRequest struct {
	Type                  *string     `json:"type" form:"type"`
	SupplierName          *string     `json:"supplier_name" form:"supplier_name"`
	SupplierContactNumber *string     `json:"supplier_contact_number" form:"supplier_contact_number"`
	SupplierAddress       *string     `json:"supplier_address" form:"supplier_address"`
	Quantity              interface{} `json:"quantity" form:"quantity"`
	Status                *bool       `json:"status" form:"status"`
}

// CreateKitHandler handles the creation of a new kit.
func CreateKitHandler(c *gin.Context) {
	// Start a new database transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
	}()

	// 1. Get the user from the context (set by AuthMiddleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Fetch the current user details within transaction
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

	// Parse and validate the request
	kitData, quantity, err := parseAndValidateKitRequest(c)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Create the Kit model
	newKit := createKitModel(kitData, quantity, user.ID)

	// Store kit in the database within transaction
	if err := tx.Create(&newKit).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Send notifications asynchronously
	go sendKitNotifications(newKit, currentUser)

	utils.JSONResponse(c, http.StatusOK, utils.MsgKitCreatedSuccessfully, nil)
}

// parseAndValidateKitRequest handles request parsing and validation
func parseAndValidateKitRequest(c *gin.Context) (KitRequest, int, error) {
	var req KitRequest
	allowedFields := []string{"type", "supplier_name", "supplier_contact_number", "supplier_address", "quantity"}

	if err := utils.ParseRequestBody(c.Request, &req, allowedFields); err != nil {
		return req, 0, err
	}

	// Trim spaces
	req.Type = strings.TrimSpace(req.Type)
	req.SupplierName = strings.TrimSpace(req.SupplierName)
	req.SupplierContactNumber = strings.TrimSpace(req.SupplierContactNumber)
	req.SupplierAddress = strings.TrimSpace(req.SupplierAddress)

	// Validate fields
	if err := validateKitRequest(req); err != nil {
		return req, 0, err
	}

	// Parse quantity
	quantity, err := parseQuantity(req.Quantity)
	if err != nil {
		return req, 0, err
	}

	return req, quantity, nil
}

// createKitModel creates a new Kit model from the request
func createKitModel(req KitRequest, quantity int, userID uint) models.Kit {
	return models.Kit{
		Type:     req.Type,
		Quantity: quantity,
		ExtraInfo: models.ExtraInfo{
			SupplierName:          req.SupplierName,
			SupplierContactNumber: req.SupplierContactNumber,
			SupplierAddress:       req.SupplierAddress,
		},
		CreatedBy: userID,
		Status:    true,
		IsDeleted: false,
	}
}

// sendKitNotifications handles the notification logic asynchronously
func sendKitNotifications(kit models.Kit, user models.User) {
	notificationUsername := fmt.Sprintf(
		"%s %s",
		user.FirstName, user.LastName,
	)
	kitNotificationMessage := fmt.Sprintf(
		"%s %s has successfully added %d %s kits.",
		user.FirstName,
		user.LastName,
		kit.Quantity,
		kit.Type,
	)
	metadata := map[string]interface{}{
		"created_by": fmt.Sprintf(
			"%s %s",
			user.FirstName, user.LastName,
		),
		"module": "Manage Inventory",
	}

	err := services.NotifyUsersByRoles(
		config.DB,
		nil,
		[]string{"super-admin", "admin"},
		"Kit Added",
		"Inventory Management",
		kitNotificationMessage,
		notificationUsername,
		"kits",
		kit.ID,
		metadata,
	)

	if err != nil {
	}
}

// Helper function to convert quantity to int
func parseQuantity(value interface{}) (int, error) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return 0, errors.New(utils.MsgQuantityCannotBeEmpty)
		}
		quantity, err := strconv.Atoi(v)
		if err != nil {
			return 0, errors.New(utils.MsgInvalidQuantityFormat)
		}
		return quantity, nil
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return 0, errors.New(utils.MsgInvalidQuantityType)
	}
}

// validateKitRequest performs validation on all fields
func validateKitRequest(req KitRequest) error {
	if req.Type == "" || req.SupplierName == "" {
		return errors.New(utils.MsgAllFieldsOfkitRequired)
	}

	// Validate Kit Type
	if !utils.IsValidKitType(req.Type) {
		return errors.New(utils.MsgInvalidKitType)
	}

	// Validate Supplier Name
	if !utils.IsValidSupplierName(req.SupplierName) {
		return errors.New(utils.MsgValidationSupplierName)
	}

	// Validate Contact Number only if provided
	if req.SupplierContactNumber != "" && !utils.IsValidContactNumber(req.SupplierContactNumber) {
		return errors.New(utils.MsgValidationContactNumber)
	}

	// Validate Address only if provided
	if req.SupplierAddress != "" && (len(req.SupplierAddress) < 5 || len(req.SupplierAddress) > 100) {
		return errors.New(utils.MsgValidationAddress)
	}

	// Validate Quantity
	quantity, err := parseQuantity(req.Quantity)
	if err != nil {
		return err
	}
	if quantity < 0 {
		return errors.New(utils.MsgQuantityCannotBeNegative)
	}
	if quantity > 999999 { // Add a reasonable upper limit
		return errors.New(utils.MsgQuantityExceedsMaxValue)
	}

	return nil
}

// GetKitsListHandler handles requests to fetch the kits list.
func GetKitsListHandler(c *gin.Context) {
	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "status", "type"}

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
	validSortColumns := []string{"type", "supplier_name", "supplier_address", "supplier_contact_number", "created_by", "created_at", "quantity"}
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

	// Optional 'type' with validation
	kitType := strings.ToLower(query.Get("type"))
	if kitType != "" && !utils.IsValidKitType(kitType) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidKitType, nil)
		return
	}

	// Initialize GORM query with Debug for detailed logging
	db := config.DB.Debug().
		Model(&models.Kit{}).
		Joins("JOIN users ON kits.created_by = users.id").
		Where("kits.is_deleted = ?", false)

	// Apply status filter
	if status == "active" {
		db = db.Where("kits.active_status = ?", true)
	} else if status == "inactive" {
		db = db.Where("kits.active_status = ?", false)
	}

	// Apply type filter
	if kitType != "" {
		db = db.Where("kits.type = ?", kitType)
	}

	// Apply search filter if searchText is provided
	if searchText != "" {
		searchPattern := "%" + searchText + "%"
		db = db.Where(
			"kits.extra_info ->> 'supplier_name' ILIKE ? OR CONCAT(users.first_name, ' ', users.last_name) ILIKE ? OR CAST(kits.created_at AS TEXT) ILIKE ? OR CAST(kits.quantity AS TEXT) ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
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

	// Apply sorting with special handling for JSONB fields
	orderClause := ""
	switch sortColumn {
	case "supplier_name":
		orderClause = fmt.Sprintf("extra_info->>'supplier_name' %s", sort)
	case "supplier_address":
		orderClause = fmt.Sprintf("extra_info->>'supplier_address' %s", sort)
	case "supplier_contact_number":
		orderClause = fmt.Sprintf("extra_info->>'supplier_contact_number' %s", sort)
	default:
		orderClause = fmt.Sprintf("%s %s", sortColumn, sort)
	}
	db = db.Order(orderClause)

	// Apply pagination
	offset := (page - 1) * perPage
	db = db.Limit(perPage).Offset(offset)

	// Fetch records
	var kits []models.Kit
	if err := db.Find(&kits).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Fetch associated users separately
	userIDs := make([]uint, len(kits))
	for i, kit := range kits {
		userIDs[i] = kit.CreatedBy
	}

	var users []models.User
	if err := config.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Create a map of user IDs to users for quick lookup
	userMap := make(map[uint]models.User)
	for _, user := range users {
		userMap[user.ID] = user
	}

	// Prepare kit details with user information
	var kitDetails []KitDetail
	for _, k := range kits {
		user, exists := userMap[k.CreatedBy]
		if !exists {
			continue
		}

		detail := KitDetail{
			ID:                    k.ID,
			Type:                  k.Type,
			Quantity:              k.Quantity,
			SupplierName:          k.ExtraInfo.SupplierName,
			SupplierAddress:       k.ExtraInfo.SupplierAddress,
			SupplierContactNumber: k.ExtraInfo.SupplierContactNumber,
			Status:                k.Status,
			CreatedAt:             k.CreatedAt,
			CreatedBy: KitsUserProfile{
				ID:        user.ID,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Email:     user.Email,
			},
		}
		kitDetails = append(kitDetails, detail)
	}

	// Prepare the response
	response := KitsListResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Type:         kitType,
		Records:      kitDetails,
	}

	// Send the response
	utils.JSONResponse(c, http.StatusOK, utils.MsgKitsListFetchedSuccessfully, response)
}

// validateKitUpdateRequest validates the update request
func validateKitUpdateRequest(req KitUpdateRequest) error {
	// Validate Type if provided
	if req.Type != nil {
		if *req.Type == "" {
			return errors.New(utils.MsgTypeCannotBeEmptyIfProvided)
		}
		if !utils.IsValidKitType(*req.Type) {
			return errors.New(utils.MsgInvalidKitType)
		}
	}

	// Validate SupplierName if provided
	if req.SupplierName != nil {
		if *req.SupplierName == "" {
			return errors.New(utils.MsgSupplierNameCannotBeEmptyIfProvided)
		}
		if !utils.IsValidSupplierName(*req.SupplierName) {
			return errors.New(utils.MsgValidationSupplierName)
		}
	}

	// Validate SupplierContactNumber if provided
	if req.SupplierContactNumber != nil && *req.SupplierContactNumber != "" {
		if !utils.IsValidContactNumber(*req.SupplierContactNumber) {
			return errors.New(utils.MsgValidationContactNumber)
		}
	}

	// Validate Address if provided
	if req.SupplierAddress != nil && *req.SupplierAddress != "" {
		if len(*req.SupplierAddress) < 5 || len(*req.SupplierAddress) > 100 {
			return errors.New(utils.MsgValidationAddress)
		}
	}

	// Validate Quantity if provided
	if req.Quantity != nil {
		quantity, err := parseQuantity(req.Quantity)
		if err != nil {
			return err
		}
		if quantity < 0 {
			return errors.New(utils.MsgQuantityCannotBeNegative)
		}
		if quantity > 999999 {
			return errors.New(utils.MsgQuantityExceedsMaxValue)
		}
	}

	return nil
}

// UpdateKitHandler handles PATCH requests to update kit details
func UpdateKitHandler(c *gin.Context) {
	// Get user from context for notification
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Get kit ID from URL parameters
	kitID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidKitID, nil)
		return
	}

	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
	}()

	// Fetch current user details
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

	// Parse and validate request
	var req KitUpdateRequest
	allowedFields := []string{"type", "supplier_name", "supplier_contact_number", "supplier_address", "quantity", "status"}

	if err := utils.ParseRequestBody(c.Request, &req, allowedFields); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Trim spaces for string fields
	trimKitUpdateRequestStrings(&req)

	// Validate the request
	if err := validateKitUpdateRequest(req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Fetch existing kit
	var kit models.Kit
	if err := tx.Where("id = ? AND is_deleted = ?", kitID, false).First(&kit).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusNotFound, utils.MsgKitNotFound, nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Store old values for notification
	oldKit := kit

	// Update kit fields
	updateKitFields(&kit, req)

	// Save the updates
	if err := tx.Save(&kit).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Send notifications asynchronously
	go sendKitUpdateNotifications(oldKit, kit, currentUser)

	utils.JSONResponse(c, http.StatusOK, utils.MsgKitUpdatedSuccessfully, nil)
}

// trimKitUpdateRequestStrings trims spaces for string fields in the request
func trimKitUpdateRequestStrings(req *KitUpdateRequest) {
	if req.Type != nil {
		*req.Type = strings.TrimSpace(*req.Type)
	}
	if req.SupplierName != nil {
		*req.SupplierName = strings.TrimSpace(*req.SupplierName)
	}
	if req.SupplierContactNumber != nil {
		*req.SupplierContactNumber = strings.TrimSpace(*req.SupplierContactNumber)
	}
	if req.SupplierAddress != nil {
		*req.SupplierAddress = strings.TrimSpace(*req.SupplierAddress)
	}
}

func updateKitFields(kit *models.Kit, req KitUpdateRequest) {
	if req.Type != nil {
		kit.Type = *req.Type
	}
	if req.Quantity != nil {
		quantity, _ := parseQuantity(req.Quantity) // Already validated
		kit.Quantity = quantity
	}
	if req.Status != nil {
		kit.Status = *req.Status
	}

	// Update ExtraInfo fields
	extraInfo := kit.ExtraInfo
	if req.SupplierName != nil {
		extraInfo.SupplierName = *req.SupplierName
	}
	if req.SupplierContactNumber != nil {
		extraInfo.SupplierContactNumber = *req.SupplierContactNumber
	}
	if req.SupplierAddress != nil {
		extraInfo.SupplierAddress = *req.SupplierAddress
	}
	kit.ExtraInfo = extraInfo
}

func sendKitUpdateNotifications(oldKit models.Kit, newKit models.Kit, user models.User) {
	changes := getKitChanges(oldKit, newKit)
	if len(changes) == 0 {
		return
	}
	notificationUsername := fmt.Sprintf(
		"%s %s",
		user.FirstName, user.LastName,
	)

	message := fmt.Sprintf(
		"%s %s has updated the %s kit.%s",
		user.FirstName,
		user.LastName,
		oldKit.Type,
		strings.Join(changes, ". "),
	)
	metadata := map[string]interface{}{
		"created_by": fmt.Sprintf(
			"%s %s",
			user.FirstName, user.LastName,
		),
		"module": "Manage Inventory",
	}
	err := services.NotifyUsersByRoles(
		config.DB,
		nil,
		[]string{"super-admin", "admin"},
		"Kit Updated",
		"Inventory Management",
		message,
		notificationUsername,
		"kits",
		newKit.ID,
		metadata,
	)

	if err != nil {

	}
}

func getKitChanges(oldKit models.Kit, newKit models.Kit) []string {
	var changes []string

	if oldKit.Type != newKit.Type {
		changes = append(changes, fmt.Sprintf("Type changed from %s to %s", oldKit.Type, newKit.Type))
	}
	if oldKit.Quantity != newKit.Quantity {
		changes = append(changes, fmt.Sprintf("Quantity changed from %d to %d", oldKit.Quantity, newKit.Quantity))
	}
	if oldKit.Status != newKit.Status {
		changes = append(changes, fmt.Sprintf("Status changed from %v to %v", oldKit.Status, newKit.Status))
	}
	if oldKit.ExtraInfo.SupplierName != newKit.ExtraInfo.SupplierName {
		changes = append(changes, fmt.Sprintf("Supplier changed from %s to %s", oldKit.ExtraInfo.SupplierName, newKit.ExtraInfo.SupplierName))
	}
	if oldKit.ExtraInfo.SupplierContactNumber != newKit.ExtraInfo.SupplierContactNumber {
		changes = append(changes, "Supplier contact number updated")
	}
	if oldKit.ExtraInfo.SupplierAddress != newKit.ExtraInfo.SupplierAddress {
		changes = append(changes, "Supplier address updated")
	}

	return changes
}

// DeleteKitHandler handles the soft deletion of kits
func DeleteKitHandler(c *gin.Context) {
	// Get user from context for notification
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Get kit ID from URL parameters
	kitID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidKitID, nil)
		return
	}

	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
	}()

	// Fetch current user details
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

	// Check if kit exists and is not already deleted
	var kit models.Kit
	if err := tx.Where("id = ? AND is_deleted = ?", kitID, false).First(&kit).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusNotFound, utils.MsgKitAlreadyDeleted, nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Perform soft delete
	updates := map[string]interface{}{
		"is_deleted": true,
		"status":     false,
	}

	if err := tx.Model(&kit).Updates(updates).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Send notifications asynchronously
	go sendKitDeleteNotifications(kit, currentUser)

	utils.JSONResponse(c, http.StatusOK, utils.MsgKitDeletedSuccessfully, nil)
}

func sendKitDeleteNotifications(kit models.Kit, user models.User) {
	notificationUsername := fmt.Sprintf(
		"%s %s",
		user.FirstName, user.LastName,
	)

	message := fmt.Sprintf(
		"%s %s has deleted %d %s kits.",
		user.FirstName,
		user.LastName,
		kit.Quantity,
		kit.Type,
	)
	metadata := map[string]interface{}{
		"created_by": fmt.Sprintf(
			"%s %s",
			user.FirstName, user.LastName,
		),
		"module": "Manage Inventory",
	}

	err := services.NotifyUsersByRoles(
		config.DB,
		nil,
		[]string{"super-admin", "admin"},
		"Kit Deleted",
		"Inventory Management",
		message,
		notificationUsername,
		"kits",
		kit.ID,
		metadata,
	)

	if err != nil {
	}
}

// GetKitsQuantitySummaryHandler handles requests to get the total quantity of blood, saliva, and combined kits
func GetKitsQuantitySummaryHandler(c *gin.Context) {
	// Initialize GORM query to fetch quantities, ignoring deleted kits
	db := config.DB.Debug().
		Model(&models.Kit{}).
		Where("kits.is_deleted = ?", false)

	// Initialize total quantities for different kit types
	var bloodQuantity, salivaQuantity, combinedQuantity int

	// Get the total quantities by type (blood, saliva) and the combined total
	err := db.Select("SUM(quantity) as quantity, type").
		Group("type").
		Find(&[]struct {
			Quantity int    `json:"quantity"`
			Type     string `json:"type"`
		}{}).Error

	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// After fetching, we need to update bloodQuantity and salivaQuantity specifically
	// If the `type` is blood or saliva, assign the quantity to the respective variable
	var kitCounts []struct {
		Quantity int    `json:"quantity"`
		Type     string `json:"type"`
	}

	// Execute the query and store the result in the kitCounts slice
	err = db.Select("SUM(quantity) as quantity, type").
		Group("type").
		Find(&kitCounts).Error

	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Iterate over the result and assign quantities based on the type
	for _, kitCount := range kitCounts {
		switch kitCount.Type {
		case "blood":
			bloodQuantity = kitCount.Quantity
		case "saliva":
			salivaQuantity = kitCount.Quantity
		}
	}

	// Calculate combined total quantity (blood + saliva)
	combinedQuantity = bloodQuantity + salivaQuantity

	// Prepare the response structure
	quantitySummary := struct {
		BloodQuantity    int `json:"blood_quantity"`
		SalivaQuantity   int `json:"saliva_quantity"`
		CombinedQuantity int `json:"combined_quantity"`
	}{
		BloodQuantity:    bloodQuantity,
		SalivaQuantity:   salivaQuantity,
		CombinedQuantity: combinedQuantity,
	}

	// Send the response
	utils.JSONResponse(c, http.StatusOK, utils.MsgTotalKitsQunatitiesFetched, quantitySummary)
}
