// models/qunatity_discounts.go

package models

import (
	"time"

	"gorm.io/gorm"
)

// QuantityDiscounts model
type QuantityDiscounts struct {
	ID                 uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Quantity           int            `gorm:"type:int;not null" json:"quantity" validate:"required,min=1"`
	DiscountPercentage float64        `gorm:"type:decimal(10,2);not null" json:"discount_percentage" validate:"required,gt=0"`
	IsDeleted          bool           `gorm:"default:false" json:"is_deleted"`
	CreatedAt          time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}
