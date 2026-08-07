package services

import (
	"fmt"
	"theransticslabs/m/models"

	"gorm.io/gorm"
)

// GetProductDiscount fetches the discount from the QuantityDiscounts table
func GetProductDiscount(tx *gorm.DB, quantity int) (float64, error) {
	var discount models.QuantityDiscounts

	// Fetch the discount if the quantity is greater than or equal to the discount quantity
	err := tx.Where("quantity <= ? AND is_deleted = ?", quantity, false).
		Order("quantity DESC"). // Order by quantity DESC to get the highest discount for the quantity
		First(&discount).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// No discount found for the given quantity
			return 0, nil
		}
		return 0, fmt.Errorf("failed to fetch discount: %v", err)
	}

	// Return the discount percentage
	return discount.DiscountPercentage, nil
}
