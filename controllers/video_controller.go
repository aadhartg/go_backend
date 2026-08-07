package controllers

import (
	"fmt"
	"io"
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
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type VideoResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
	MimeType    string    `json:"mime_type"`
	Duration    float64   `json:"duration"`
	UploadedBy  uint      `json:"uploaded_by"`
	UserName    string    `json:"user_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// VideosResponse represents the structured response for videos list.
type VideosResponse struct {
	Page         int             `json:"page"`
	PerPage      int             `json:"per_page"`
	Sort         string          `json:"sort"`
	SortColumn   string          `json:"sort_column"`
	SearchText   string          `json:"search_text"`
	TotalRecords int64           `json:"total_records"`
	TotalPages   int             `json:"total_pages"`
	Records      []VideoResponse `json:"records"`
}

// UploadVideo handles video file uploads
func UploadVideo(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can upload videos", nil)
		return
	}

	// Get and validate title
	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Video title is required", nil)
		return
	}

	// Check if title already exists
	var existingVideo models.Video
	if err := config.DB.Where("title = ? AND is_deleted = ?", title, false).First(&existingVideo).Error; err == nil {
		utils.JSONResponse(c, http.StatusConflict, "A video with this title already exists", nil)
		return
	} else if err != gorm.ErrRecordNotFound {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to check video title", nil)
		return
	}

	// Parse multipart form
	err := c.Request.ParseMultipartForm(utils.MaxVideoSize)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "File too large or invalid form", nil)
		return
	}

	// Get file from request
	file, handler, err := c.Request.FormFile("video")
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "No video file uploaded", nil)
		return
	}
	defer file.Close()

	// Validate file type
	mimeType := handler.Header.Get("Content-Type")
	if !strings.Contains(utils.AllowedVideoMime, mimeType) {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid video format. Allowed formats: MP4, MOV, AVI, MKV", nil)
		return
	}

	// Check file size
	if handler.Size > utils.MaxVideoSize {
		utils.JSONResponse(c, http.StatusBadRequest, "File size exceeds 100MB limit", nil)
		return
	}

	// Generate unique filename
	currentDate := time.Now().Format("20060102")
	uniqueID := utils.GenerateUniqueID()
	fileExt := filepath.Ext(handler.Filename)
	newFilename := fmt.Sprintf("%s_%s%s", currentDate, uniqueID, fileExt)

	// Define upload directory
	uploadDir := utils.VideoUploadPath
	filePath := filepath.Join(uploadDir, newFilename)

	// Create the upload directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create upload directory", nil)
		return
	}

	// Create new file
	dst, err := os.Create(filePath)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create file", nil)
		return
	}
	defer dst.Close()

	// Copy uploaded file contents
	if _, err := io.Copy(dst, file); err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save file", nil)
		return
	}

	// Create video record in database
	video := models.Video{
		Title:       title,
		Description: c.PostForm("description"),
		FilePath:    newFilename,
		FileSize:    handler.Size,
		MimeType:    mimeType,
		Duration:    0, // Todo: Need to create functionality to manage this. (pending)
		UploadedBy:  user.ID,
	}

	if err := config.DB.Create(&video).Error; err != nil {
		// Delete the uploaded file if database operation fails
		os.Remove(filePath)
		fmt.Print("@1 Error while uploadin the video: ", err)
		fmt.Print("@2 Error while uploadin the video: ", video)
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save video record", nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, "Video uploaded successfully", video)
}

// GetVideos retrieves a list of videos with filtering, pagination, and sorting
func GetVideos(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can view videos", nil)
		return
	}

	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text"}

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
	validSortColumns := []string{"title", "file_size", "created_at", "updated_at", "file_path"}
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

	// Initialize GORM query
	db := config.DB.Model(&models.Video{}).
		Where("is_deleted = ?", false)

	// Debug: Check total videos in database (before any search filter)
	var totalVideosInDB int64
	config.DB.Model(&models.Video{}).Where("is_deleted = ?", false).Count(&totalVideosInDB)

	// Apply search filter
	if searchText != "" {
		// Use the same helper to build the WHERE clause
		// Map: title -> title, description -> description, name -> title (for consistency)
		// The date filtering is handled automatically by the utility function
		whereClause, args := utils.BuildDateSearchQueryWithColumns(
			db, // base query (gorm.DB)
			searchText,
			dateResult,
			"videos",      // table alias
			"title",       // field to match title text
			"description", // field to match description text
			"title",       // field to match name (using title as fallback)
		)

		// Apply the resulting WHERE clause
		db = db.Where(whereClause, args...)
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

	// Apply sorting
	db = db.Order(sortColumn + " " + sort)

	// Apply pagination
	offset := (page - 1) * perPage
	db = db.Limit(perPage).Offset(offset)

	// Fetch records with user information
	var videos []models.Video
	if err := db.Preload("User").Find(&videos).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchRecords, nil)
		return
	}

	// Prepare video responses
	var videoResponses []VideoResponse
	for _, v := range videos {
		fullName := fmt.Sprintf("%s %s", v.User.FirstName, v.User.LastName)
		videoResponses = append(videoResponses, VideoResponse{
			ID:          v.ID,
			Title:       v.Title,
			Description: v.Description,
			FilePath:    v.FilePath,
			FileSize:    v.FileSize,
			MimeType:    v.MimeType,
			Duration:    v.Duration,
			UploadedBy:  v.UploadedBy,
			UserName:    fullName,
			CreatedAt:   v.CreatedAt,
			UpdatedAt:   v.UpdatedAt,
		})
	}

	// Prepare the response
	response := VideosResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      videoResponses,
	}

	// Send the response
	utils.JSONResponse(c, http.StatusOK, "Videos fetched successfully", response)
}

// GetVideoByTitle retrieves a single video by its title
func GetVideoByTitle(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can view videos", nil)
		return
	}

	title := c.Param("title")
	if title == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Video title is required", nil)
		return
	}

	var video models.Video
	if err := config.DB.Preload("User").Where("title = ? AND is_deleted = ?", title, false).First(&video).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusNotFound, "Video not found", nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch video", nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, "Video fetched successfully", video)
}

// DeleteVideo deletes a video if it's not attached to any consent form
func DeleteVideo(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can delete videos", nil)
		return
	}

	videoID := c.Param("id")
	if videoID == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Video ID is required", nil)
		return
	}

	// Fetch video including soft-deleted records if needed
	var video models.Video
	if err := config.DB.Unscoped().First(&video, videoID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusNotFound, "Video not found", nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch video", nil)
		return
	}

	// Check if video is attached to any consent form
	var consentForm models.EConsentForm
	if err := config.DB.
		Where("video_id = ? AND is_deleted = ?", videoID, false).
		First(&consentForm).Error; err == nil {

		utils.JSONResponse(c, http.StatusConflict, "Cannot delete video as it is attached to a consent form", nil)
		return

	} else if err != gorm.ErrRecordNotFound {

		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to check video usage", nil)
		return
	}

	// Delete the video file
	filePath := filepath.Join(utils.VideoUploadPath, video.FilePath)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to delete video file", nil)
		return
	}

	// Permanently delete record from database
	if err := config.DB.Unscoped().Delete(&video).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to delete video record", nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, "Video deleted successfully", nil)
}

// EditVideo updates video details and optionally replaces the video file
func EditVideo(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Check if user has admin or super-admin role
	if user.RoleID != 1 && user.RoleID != 2 {
		utils.JSONResponse(c, http.StatusForbidden, "Only super-admin and admin users can edit videos", nil)
		return
	}

	videoID := c.Param("id")
	if videoID == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Video ID is required", nil)
		return
	}

	// Get and validate new title
	newTitle := strings.TrimSpace(c.PostForm("title"))
	if newTitle == "" {
		utils.JSONResponse(c, http.StatusBadRequest, "Video title is required", nil)
		return
	}

	// Check if new title already exists (excluding current video)
	var existingVideo models.Video
	if err := config.DB.Where("title = ? AND id != ? AND is_deleted = ?", newTitle, videoID, false).First(&existingVideo).Error; err == nil {
		utils.JSONResponse(c, http.StatusConflict, "A video with this title already exists", nil)
		return
	} else if err != gorm.ErrRecordNotFound {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to check video title", nil)
		return
	}

	// Get the video
	var video models.Video
	if err := config.DB.First(&video, videoID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(c, http.StatusNotFound, "Video not found", nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch video", nil)
		return
	}

	// Handle file upload if a new video is provided
	updates := map[string]interface{}{
		"title":       newTitle,
		"description": c.PostForm("description"),
	}

	// Check if a new video file was uploaded
	file, handler, err := c.Request.FormFile("video")
	if err == nil {
		defer file.Close()

		// Validate file type
		mimeType := handler.Header.Get("Content-Type")
		if !strings.Contains(utils.AllowedVideoMime, mimeType) {
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid video format. Allowed formats: MP4, MOV, AVI, MKV", nil)
			return
		}

		// Generate unique filename
		currentDate := time.Now().Format("20060102")
		uniqueID := utils.GenerateUniqueID()
		fileExt := filepath.Ext(handler.Filename)
		newFilename := fmt.Sprintf("%s_%s%s", currentDate, uniqueID, fileExt)

		// Define upload directory
		uploadDir := utils.VideoUploadPath
		newFilePath := filepath.Join(uploadDir, newFilename)

		// Create new file
		dst, err := os.Create(newFilePath)
		if err != nil {
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create new video file", nil)
			return
		}
		defer dst.Close()

		// Copy uploaded file contents
		if _, err := io.Copy(dst, file); err != nil {
			os.Remove(newFilePath) // Clean up the new file if copy fails
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to save new video file", nil)
			return
		}

		// Delete the old video file
		oldFilePath := filepath.Join(uploadDir, video.FilePath)
		if err := os.Remove(oldFilePath); err != nil && !os.IsNotExist(err) {
			os.Remove(newFilePath) // Clean up the new file if old file deletion fails
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to delete old video file", nil)
			return
		}

		// Update video record with new file information
		updates["file_path"] = newFilename
		updates["file_size"] = handler.Size
		updates["mime_type"] = mimeType
		updates["duration"] = 0 // Reset duration as it needs to be recalculated
	}

	// Update video details
	if err := config.DB.Model(&video).Updates(updates).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to update video", nil)
		return
	}

	// Fetch the updated video with user information
	if err := config.DB.Preload("User").First(&video, videoID).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch updated video", nil)
		return
	}

	// response
	fullName := fmt.Sprintf("%s %s", video.User.FirstName, video.User.LastName)
	response := VideoResponse{
		ID:          video.ID,
		Title:       video.Title,
		Description: video.Description,
		FilePath:    video.FilePath,
		FileSize:    video.FileSize,
		MimeType:    video.MimeType,
		Duration:    video.Duration,
		UploadedBy:  video.UploadedBy,
		UserName:    fullName,
		CreatedAt:   video.CreatedAt,
		UpdatedAt:   video.UpdatedAt,
	}

	utils.JSONResponse(c, http.StatusOK, "Video updated successfully", response)
}
