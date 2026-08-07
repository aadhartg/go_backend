// models/barcode.go

package models

import (
	"time"

	"gorm.io/gorm"
)

// Barcode model
type Barcode struct {
	ID              uint              `gorm:"primaryKey;autoIncrement;not null" json:"id"`
	BarcodeNumber   string            `gorm:"type:varchar(50);unique;not null" json:"barcode_number" validate:"required"`
	OrderID         uint              `gorm:"not null" json:"order_id" validate:"required"`
	Order           Order             `gorm:"foreignKey:OrderID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"order,omitempty"`
	CreatedAt       time.Time         `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsDeleted       bool              `gorm:"default:false" json:"is_deleted"`
	DeletedAt       gorm.DeletedAt    `gorm:"index" json:"-"`
	KitRegistration []KitRegistration `gorm:"foreignKey:KitBarcodeOrderID" json:"kit_registration,omitempty"`
}
