// `seeds/quantity_discounts_seeder.go`

package seeds

import (
	"log"

	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"gorm.io/gorm"
)

// SeedQuantityDiscounts inserts predefined quantity discount entries into the quantity_discounts table.
// It ensures that discounts are not duplicated by checking for existing entries.
func SeedQuantityDiscounts() {
	// Define the quantity discount to be seeded
	quantityDiscount := models.QuantityDiscounts{
		Quantity:           100,
		DiscountPercentage: 15,
	}

	// Check if the discount for the specified quantity already exists
	var existingDiscount models.QuantityDiscounts
	result := config.DB.Where("quantity = ?", quantityDiscount.Quantity).First(&existingDiscount)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Discount does not exist, create it
			if err := config.DB.Create(&quantityDiscount).Error; err != nil {
				log.Printf(utils.MsgFailedToCreateDiscount, quantityDiscount.Quantity, quantityDiscount.DiscountPercentage, err)
			} else {
				log.Printf(utils.MsgDiscountCreated, quantityDiscount.Quantity, quantityDiscount.DiscountPercentage)
			}
		} else {
			// An unexpected error occurred
			log.Printf(utils.MsgFailedToCheckDiscount, quantityDiscount.Quantity, result.Error)
		}
	} else {
		// Discount already exists
		log.Printf(utils.MsgDiscountAlreadyExists, quantityDiscount.Quantity)
	}

	log.Println(utils.MsgDiscountsSeededSuccessfully)
}
