// quantity_discount_controller
package controllers

import (
	"net/http"
	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

// FetchQuantityDiscountList handles the API request to fetch the list of quantity discounts.
func FetchQuantityDiscountList(c *gin.Context) {
	// Define a slice to hold the required fields
	var quantityDiscounts []struct {
		ID                 uint    `json:"id"`
		Quantity           int     `json:"quantity"`
		DiscountPercentage float64 `json:"discount_percentage"`
	}

	// Query to fetch all non-deleted quantity discounts
	if err := config.DB.Debug().
		Model(&models.QuantityDiscounts{}).
		Where("is_deleted = ?", false).
		Order("created_at DESC").
		Find(&quantityDiscounts).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToFetchQuantityAndDiscount, nil)
		return
	}

	// Send successful response
	utils.JSONResponse(c, http.StatusOK, utils.MsgSuccessToFetchQuantityAndDiscount, quantityDiscounts)
}
