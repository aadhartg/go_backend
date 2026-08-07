// models/kit.go
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ExtraInfo represents the structure for supplier information stored in the extra_info JSON field.
type ExtraInfo struct {
	SupplierName          string `json:"supplier_name" validate:"required,max=50"`           // Supplier name (required and max 50 characters)
	SupplierContactNumber string `json:"supplier_contact_number" validate:"required,max=15"` // Supplier contact number (required and max 15 characters)
	SupplierAddress       string `json:"supplier_address" validate:"required,max=255"`       // Supplier address (required and max 255 characters)
}

// Value makes ExtraInfo implement the driver.Valuer interface.
func (ei ExtraInfo) Value() (driver.Value, error) {
	return json.Marshal(ei)
}

// Scan makes ExtraInfo implement the sql.Scanner interface.
func (ei *ExtraInfo) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, &ei)
}

// Kit represents the kits table in the database.
type Kit struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Type          string         `gorm:"type:varchar(10);not null" json:"type" validate:"required,max=10"`                                                  // Kit type (required and max 10 characters)
	Quantity      int            `gorm:"not null;default:0" json:"quantity" validate:"required,min=0"`                                                      // Quantity (required and should not be negative)
	ExtraInfo     ExtraInfo      `gorm:"type:json" json:"extra_info"`                                                                                       // Supplier-related extra information (stored as JSON)
	CreatedBy     uint           `gorm:"not null" json:"created_by" validate:"required"`                                                                    // ID of the user who created the kit
	CreatedByUser User           `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"created_by_user,omitempty"` // User associated with the creation
	Status        bool           `gorm:"default:true" json:"status" validate:"required"`                                                                    // Status of the kit (default is true)
	IsDeleted     bool           `gorm:"default:false" json:"is_deleted"`                                                                                   // Soft delete flag (default is false)
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`                                                                                  // Timestamp for when the kit was created
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`                                                                                  // Timestamp for when the kit was last updated
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`                                                                                                    // Timestamp for soft deletion (hidden in responses)
}
