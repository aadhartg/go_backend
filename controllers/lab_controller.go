package controllers

import (
	"net/http"
	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

type LabResponse struct {
	ID         uint   `json:"id"`
	LabName    string `json:"lab_name"`
	LabAddress string `json:"lab_address"`
	NHINumber  string `json:"nhi_number"`
}

func GetAllLabs(c *gin.Context) {
	var labs []LabResponse

	// Query the database with soft delete consideration
	// Query the database and select specific fields
	result := config.DB.Model(&models.Lab{}).
		Select("id, lab_name, lab_address, nhi_number").
		Where("is_deleted = ?", false).
		Find(&labs)

	if result.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to fetch labs", nil)
		return
	}

	// If no labs found, return empty array instead of null
	if len(labs) == 0 {
		utils.JSONResponse(c, http.StatusOK, "No labs found", []models.Lab{})
		return
	}

	utils.JSONResponse(c, http.StatusOK, "Labs fetched successfully", labs)
}
