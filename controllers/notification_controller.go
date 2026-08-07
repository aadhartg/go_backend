// notification_controller
package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

// NotificationResponse represents the structured response for notifications
type NotificationResponse struct {
	Page         int                `json:"page"`
	PerPage      int                `json:"per_page"`
	Sort         string             `json:"sort"`
	SortColumn   string             `json:"sort_column"`
	SearchText   string             `json:"search_text"`
	Type         string             `json:"type"`
	IsRead       *bool              `json:"is_read"`
	TotalRecords int64              `json:"total_records"`
	TotalPages   int                `json:"total_pages"`
	Records      []NotificationData `json:"records"`
	StartDate    string             `json:"start_date"`
	EndDate      string             `json:"end_date"`
}

type NotificationData struct {
	ID        uint            `json:"id"`
	Title     string          `json:"title"`
	Message   string          `json:"message"`
	Type      string          `json:"type"`
	IsRead    bool            `json:"is_read"`
	Username  string          `json:"username"`
	CreatedAt time.Time       `json:"created_at"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type MarkAsReadRequest struct {
	IsRead bool `json:"is_read" form:"is_read" validate:"required"`
}

// GetUserNotifications handles fetching notifications for a user with filtering and pagination
func GetUserNotifications(c *gin.Context) {
	// Get user from context (set by AuthMiddleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "type", "is_read", "start_date", "end_date"}

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
	validSortColumns := []string{"title", "type", "entity_type", "is_read", "created_at"}
	if val := strings.ToLower(query.Get("sort_column")); val != "" {
		if utils.StringInSlice(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortColumnParameter, nil)
			return
		}
	}

	// Optional filters
	searchText := strings.TrimSpace(query.Get("search_text"))
	notificationType := strings.TrimSpace(strings.ToLower(query.Get("type")))

	// Validate the notification type
	allowedTypes := []string{"admin management", "inventory management", "order purchase", "order management"}
	if notificationType != "" && !utils.StringInSlice(notificationType, allowedTypes) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidNotificationType, nil)
		return
	}

	// Parse is_read filter if provided
	var isRead *bool
	if val := query.Get("is_read"); val != "" {
		boolVal, err := strconv.ParseBool(val)
		if err == nil {
			isRead = &boolVal
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidIsReadParameter, nil)
			return
		}
	}

	// Parse and validate date range
	var startDate, endDate *time.Time
	if startDateStr := query.Get("start_date"); startDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidStartDate, nil)
			return
		}
		startDate = &parsedDate
	}

	if endDateStr := query.Get("end_date"); endDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidEndDate, nil)
			return
		}
		endDate = &parsedDate
		*endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second) // End of day
	}

	// Ensure valid date range
	if startDate != nil && endDate != nil && startDate.After(*endDate) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidDateRange, nil)
		return
	}

	// Build the base query
	notificationsDB := config.DB.Debug().
		Model(&models.Notification{}).
		Where("user_id = ? AND is_deleted = ?", user.ID, false)

	// Apply search filter if provided
	if searchText != "" {
		searchPattern := "%" + searchText + "%"
		notificationsDB = notificationsDB.Where(
			"title ILIKE ? OR message ILIKE ? OR type ILIKE ? OR entity_type ILIKE ? OR CAST(created_at AS TEXT) ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Apply type filter if provided
	if notificationType != "" {
		notificationsDB = notificationsDB.Where("LOWER(type) = ?", notificationType)
	}

	// Apply is_read filter if provided
	if isRead != nil {
		notificationsDB = notificationsDB.Where("is_read = ?", *isRead)
	}

	// Apply date range filter
	if startDate != nil {
		notificationsDB = notificationsDB.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		notificationsDB = notificationsDB.Where("created_at <= ?", *endDate)
	}

	// Get total records count
	var totalRecords int64
	if err := notificationsDB.Count(&totalRecords).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCountRecords, nil)
		return
	}

	// Calculate total pages
	totalPages := int((totalRecords + int64(perPage) - 1) / int64(perPage))
	if totalRecords == 0 {
		totalPages = 0
	}

	// Apply sorting and pagination
	var notifications []models.Notification
	offset := (page - 1) * perPage
	if err := notificationsDB.
		Order(sortColumn + " " + sort).
		Limit(perPage).
		Offset(offset).
		Find(&notifications).Error; err != nil {
		log.Printf("Error fetching notifications: %v", err)
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Prepare response records
	notificationData := make([]NotificationData, len(notifications))
	for i, notification := range notifications {
		notificationData[i] = NotificationData{
			ID:        notification.ID,
			Title:     notification.Title,
			Message:   notification.Message,
			Type:      notification.Type,
			IsRead:    notification.IsRead,
			Username:  notification.Username,
			Metadata:  notification.Metadata,
			CreatedAt: notification.CreatedAt,
		}
	}

	// Prepare response
	response := NotificationResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		Type:         notificationType,
		IsRead:       isRead,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      notificationData,
	}

	// Include date range in response if provided
	if startDate != nil {
		response.StartDate = startDate.Format("2006-01-02")
	}
	if endDate != nil {
		response.EndDate = endDate.Format("2006-01-02")
	}

	// Send response
	utils.JSONResponse(c, http.StatusOK, utils.MsgNotificationsFetchedSuccess, response)
}

// GetLatestNotifications handles fetching the latest 10 notifications for a user
// and checks if any of the fetched notifications are unread.
func GetLatestNotifications(c *gin.Context) {
	// Get user from context (set by AuthMiddleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Build the base query for notifications
	var notifications []models.Notification
	notificationsDB := config.DB.Debug().
		Model(&models.Notification{}).
		Where("user_id = ? AND is_deleted = ?", user.ID, false).
		Order("created_at DESC").
		Limit(10)

	// Execute the query to fetch the notifications
	if err := notificationsDB.Find(&notifications).Error; err != nil {
		log.Printf("Error fetching latest notifications: %v", err)
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Query to fetch the total count of unread notifications
	var unreadCount int64
	unreadCountDB := config.DB.Debug().
		Model(&models.Notification{}).
		Where("user_id = ? AND is_deleted = ? AND is_read = ?", user.ID, false, false)

	// Execute the query to fetch unread count
	if err := unreadCountDB.Count(&unreadCount).Error; err != nil {
		log.Printf("Error fetching unread notification count: %v", err)
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Prepare response data
	notificationData := make([]NotificationData, len(notifications))
	hasUnread := false

	for i, notification := range notifications {
		notificationData[i] = NotificationData{
			ID:        notification.ID,
			Title:     notification.Title,
			Message:   notification.Message,
			Type:      notification.Type,
			IsRead:    notification.IsRead,
			Username:  notification.Username,
			Metadata:  notification.Metadata,
			CreatedAt: notification.CreatedAt,
		}
		// Check if any notification is unread
		if !notification.IsRead {
			hasUnread = true
		}
	}

	// Response structure
	response := map[string]interface{}{
		"latest_notifications": notificationData,
		"has_unread":           hasUnread,
		"unread_count":         unreadCount,
	}

	// Send response
	utils.JSONResponse(c, http.StatusOK, utils.MsgNotificationsFetchedSuccess, response)
}

// MarkNotificationRead handles marking a specific notification as read or unread
func MarkNotificationRead(c *gin.Context) {
	// Get user from context
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Parse notification ID from the query parameters
	notificationIDStr := c.Param("id")
	if notificationIDStr == "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgNotificationIDRequired, nil)
		return
	}

	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil || notificationID <= 0 {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidNotificationID, nil)
		return
	}

	// Parse and Validate Request Body
	var req MarkAsReadRequest
	if err := utils.ParseRequestBody(c.Request, &req, []string{"is_read"}); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Check if the notification exists and belongs to the user
	var notification models.Notification
	if err := config.DB.Debug().
		Where("id = ? AND user_id = ? AND is_deleted = ?", notificationID, user.ID, false).
		First(&notification).Error; err != nil {
		utils.JSONResponse(c, http.StatusNotFound, utils.MsgNotificationNotFound, nil)
		return
	}

	// Check if the status is already the same
	if notification.IsRead == req.IsRead {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgNotificationAlreadySet, nil)
		return
	}

	// Update the `is_read` status
	notification.IsRead = req.IsRead
	if err := config.DB.Save(&notification).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToUpdateNotification, nil)
		return
	}

	// Send success response
	utils.JSONResponse(c, http.StatusOK, utils.MsgNotificationMarkedAsRead, nil)
}

// MarkAllNotificationsRead handles marking all notifications as read for a user
func MarkAllNotificationsRead(c *gin.Context) {
	// Get user from context
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if there are any notifications to mark as read
	var count int64
	if err := config.DB.Debug().
		Model(&models.Notification{}).
		Where("user_id = ? AND is_deleted = ? AND is_read = ?", user.ID, false, false).
		Count(&count).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCountUnreadNotifications, nil)
		return
	}

	if count == 0 {
		utils.JSONResponse(c, http.StatusOK, utils.MsgNoUnreadNotificationsToMarkAsRead, nil)
		return
	}

	// Mark all unread notifications as read
	if err := config.DB.Debug().
		Model(&models.Notification{}).
		Where("user_id = ? AND is_deleted = ? AND is_read = ?", user.ID, false, false).
		Update("is_read", true).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToMarkAllNotificationsAsRead, nil)
		return
	}

	// Send success response
	utils.JSONResponse(c, http.StatusOK, utils.MsgAllUnreadNotificationsMarkedAsRead, nil)
}

// DeleteUserNotifications handles soft deletion of notifications within a date range for a specific user
func DeleteUserNotifications(c *gin.Context) {
	// Get user from context (set by AuthMiddleware)
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Define allowed query parameters
	allowedFields := []string{"start_date", "end_date"}

	// Parse query parameters with default values
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidQueryParameters, nil)
		return
	}

	startDateStr := strings.TrimSpace(query.Get("start_date"))
	endDateStr := strings.TrimSpace(query.Get("end_date"))

	// Validate date range
	if startDateStr == "" || endDateStr == "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgDateRangeRequired, nil)
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidStartDateFormat, nil)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidEndDateFormat, nil)
		return
	}

	// Ensure valid date range
	if startDate.After(endDate) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidDateRange, nil)
		return
	}

	// Set time range for the date filter
	startDate = startDate.Truncate(24 * time.Hour)                                                 // Set time to 00:00:00
	endDate = endDate.Truncate(24 * time.Hour).Add(23*time.Hour + 59*time.Minute + 59*time.Second) // Set time to 23:59:59

	// Check if data exists for the given date range and user
	var count int64
	err = config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_deleted = ? AND created_at BETWEEN ? AND ?", user.ID, false, startDate, endDate).
		Count(&count).Error

	if err != nil {
		log.Printf("Error checking notifications: %v", err)
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	if count == 0 {
		utils.JSONResponse(c, http.StatusNotFound, utils.MsgNoNotificationsFoundForDateRange, nil)
		return
	}

	// Soft delete notifications
	err = config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_deleted = ? AND created_at BETWEEN ? AND ?", user.ID, false, startDate, endDate).
		Update("is_deleted", true).Error

	if err != nil {
		log.Printf("Error soft deleting notifications: %v", err)
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToDeleteNotifications, nil)
		return
	}

	// Success response
	utils.JSONResponse(c, http.StatusOK, utils.MsgNotificationsDeletedSuccess, nil)
}

// DeleteNotification handles the deletion of a specific notification for a user
func DeleteNotification(c *gin.Context) {
	// Get user from context
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Parse notification ID from the query parameters
	notificationIDStr := c.Param("id")
	if notificationIDStr == "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgNotificationIDRequired, nil)
		return
	}

	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil || notificationID <= 0 {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidNotificationID, nil)
		return
	}

	// Check if the notification exists and belongs to the user
	var notification models.Notification
	if err := config.DB.Debug().
		Where("id = ? AND user_id = ? AND is_deleted = ?", notificationID, user.ID, false).
		First(&notification).Error; err != nil {
		utils.JSONResponse(c, http.StatusNotFound, utils.MsgNotificationNotFound, nil)
		return
	}

	// Soft delete the notification
	notification.IsDeleted = true
	if err := config.DB.Save(&notification).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.InternalServerError, nil)
		return
	}

	// Send success response
	utils.JSONResponse(c, http.StatusOK, utils.NotificationDeletedMessage, nil)
}
