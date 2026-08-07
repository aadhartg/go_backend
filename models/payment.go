// models/payment.go

package models

import (
	"time"

	"gorm.io/gorm"
)

// Payment model
type Payment struct {
	ID              uint           `gorm:"primaryKey;autoIncrement;not null" json:"id"`
	OrderID         uint           `gorm:"not null" json:"order_id" validate:"required"`
	Order           Order          `gorm:"foreignKey:OrderID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"order,omitempty"`
	PaymentStatus   string         `gorm:"type:varchar(50);not null" json:"dna_payment_status" validate:"required,oneof=Pending Completed Failed"`
	TransactionID   string         `gorm:"type:varchar(100);not null;unique" json:"transaction_id" validate:"required"`
	Amount          float64        `gorm:"type:decimal(10,2);not null" json:"amount" validate:"required,gt=0"`
	ProductGstPrice float64        `gorm:"type:decimal(10,2);not null" json:"product_gst_price" validate:"required,gt=0"`
	ProductDiscount float64        `gorm:"type:decimal(10,2);default:0;not null" json:"product_discount" validate:"required"`
	Invoices        []Invoice      `gorm:"foreignKey:PaymentID" json:"invoices,omitempty"`
	CreatedAt       time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsDeleted       bool           `gorm:"default:false" json:"is_deleted"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}
