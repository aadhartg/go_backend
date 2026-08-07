// models/kit_registration.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// KitRegistration represents the "kit_registrations" table in the database.
type KitRegistration struct {
	ID                uint           `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	KitCustomerID     uint           `gorm:"not null;column:kit_customer_id" json:"kit_customer_id" validate:"required"`                              // Foreign key for Customer
	Customer          Customer       `gorm:"foreignKey:KitCustomerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"customer"`    // Related Customer
	KitOrderID        uint           `gorm:"not null;column:kit_order_id" json:"kit_order_id" validate:"required"`                                    // Foreign key for Order
	Order             Order          `gorm:"foreignKey:KitOrderID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"order"`          // Related Order
	KitBarcodeOrderID uint           `gorm:"not null;column:kit_barcode_order_id" json:"kit_barcode_order_id" validate:"required"`                    // Foreign key for Barcode
	Barcode           Barcode        `gorm:"foreignKey:KitBarcodeOrderID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"barcode"` // Related Barcode
	PatientFirstName  string         `gorm:"type:text;not null;column:patient_first_name" json:"patient_first_name" validate:"required,max=255"`
	PatientLastName   string         `gorm:"type:text;not null;column:patient_last_name" json:"patient_last_name" validate:"required,max=255"`
	PatientEmail      string         `gorm:"type:text;not null;column:patient_email" json:"patient_email" validate:"required,email,max=255"`
	PatientGender     string         `gorm:"type:text;not null;column:patient_gender" json:"patient_gender" validate:"required,max=255"`
	PatientAge        string         `gorm:"type:text;not null;column:patient_age" json:"patient_age" validate:"required,max=255"`
	KitStatus         string         `gorm:"type:varchar(50);not null;default:Not-Received" json:"dna_kit_status" validate:"required,oneof=Not-Received Received Reject Send"`
	Reason            string         `gorm:"type:text" json:"reason"`
	FilePath          string         `gorm:"type:varchar(255);column:file_path" json:"file_path"` // New field to store file path
	LabID             uint           `gorm:"null;column:lab_id" json:"lab_id" validate:"required"`
	IsClinicInform    bool           `gorm:"type:boolean;column:is_clinic_inform" json:"is_clinic_inform"`
	IsDeleted         bool           `gorm:"default:false;column:is_deleted" json:"is_deleted"`  // Soft delete flag
	CreatedAt         time.Time      `gorm:"autoCreateTime;column:created_at" json:"created_at"` // Creation timestamp
	UpdatedAt         time.Time      `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"` // Update timestamp
	DeletedAt         gorm.DeletedAt `gorm:"index;column:deleted_at" json:"-"`                   // Soft deletion timestamp
}
