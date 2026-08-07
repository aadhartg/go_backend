// models/payment.go

package models

import (
	"time"

	"gorm.io/gorm"
)

// Invoice model
type Invoice struct {
	ID              uint           `gorm:"primaryKey;autoIncrement;not null" json:"id"`
	PaymentID       uint           `gorm:"not null" json:"payment_id" validate:"required"`
	Payment         Payment        `gorm:"foreignKey:PaymentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"payment,omitempty"`
	InvoiceLink     string         `gorm:"type:varchar(255);not null" json:"invoice_link" validate:"required,url"`
	Price           float64        `gorm:"type:decimal(10,2);not null" json:"price" validate:"required,gt=0"`
	ProductGstPrice float64        `gorm:"type:decimal(10,2);not null" json:"product_gst_price" validate:"required,gt=0"`
	ProductDiscount float64        `gorm:"type:decimal(10,2);default:0;not null" json:"product_discount" validate:"required"`
	InvoiceID       string         `gorm:"type:varchar(100);not null;unique" json:"invoice_id" validate:"required"`
	CreatedAt       time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsDeleted       bool           `gorm:"default:false" json:"is_deleted"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}
