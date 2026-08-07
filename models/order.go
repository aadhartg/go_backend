package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// StatusEntry defines a single order status change with a timestamp
type StatusEntry struct {
	Status    string    `json:"status"`    // e.g., Pending, Processing, etc.
	Timestamp time.Time `json:"timestamp"` // Time when the status was updated
}

// Order model
type Order struct {
	ID                 uint              `gorm:"primaryKey;autoIncrement;not null" json:"id"`
	OrderNumber        string            `gorm:"type:varchar(100);not null;unique" json:"order_number" validate:"required"`
	CustomerID         uint              `gorm:"not null" json:"customer_id" validate:"required"`
	Customer           Customer          `gorm:"foreignKey:CustomerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"customer,omitempty"`
	ProductName        string            `gorm:"type:varchar(100);not null" json:"product_name" validate:"required"`
	ProductDescription string            `gorm:"type:text" json:"product_description"`
	ProductImage       string            `gorm:"type:text" json:"product_image"`
	ProductPrice       float64           `gorm:"type:decimal(10,2);not null" json:"product_price" validate:"required,gt=0"`
	ProductGstPrice    float64           `gorm:"type:decimal(10,2);not null" json:"product_gst_price" validate:"required,gt=0"`
	ProductDiscount    float64           `gorm:"type:decimal(10,2);default:0;not null" json:"product_discount" validate:"required"`
	Quantity           int               `gorm:"type:int;not null" json:"quantity" validate:"required,min=1"`
	TotalPrice         float64           `gorm:"type:decimal(10,2);not null" json:"total_price" validate:"required,gt=0"`
	PaymentStatus      string            `gorm:"type:varchar(50);not null" json:"dna_payment_status" validate:"required,oneof=Pending Completed Failed"`
	OrderStatus        string            `gorm:"type:varchar(50);not null" json:"dna_order_status" validate:"required,oneof=Pending Processing Shipped Dispatched Delivered Cancelled"`
	OrderStatusHistory datatypes.JSON    `gorm:"type:jsonb" json:"order_status_history"` // New field for status history
	TrackingID         string            `gorm:"type:varchar(100);" json:"tracking_id"`
	Payments           []Payment         `gorm:"foreignKey:OrderID" json:"payments,omitempty"`
	Barcodes           []Barcode         `gorm:"foreignKey:OrderID" json:"barcodes,omitempty"` // New association
	KitRegistration    []KitRegistration `gorm:"foreignKey:KitOrderID" json:"kit_registration,omitempty"`
	CreatedAt          time.Time         `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time         `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsDeleted          bool              `gorm:"default:false" json:"is_deleted"`
	DeletedAt          gorm.DeletedAt    `gorm:"index" json:"-"`
}
