// models/user.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// User model
type User struct {
	ID          				 						 uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	FirstName    										 string         `gorm:"type:varchar(50);not null" json:"first_name" validate:"required,max=50"`           // First name is required and max 50 characters
	LastName     										 string         `gorm:"type:varchar(50);null" json:"last_name" validate:"max=50"`                         // Last name is optional but limited to 50 characters
	Email        										 string         `gorm:"type:varchar(100);unique;not null" json:"email" validate:"required,email,max=100"` // Email is required, valid email format, and max 100 characters
	HashPassword 										 string         `gorm:"type:varchar(255);null" json:"hash_password" validate:"required,max=255"`          // Password hash is required and max 255 characters
	RoleID       										 uint           `gorm:"not null" json:"role_id" validate:"required"`                                      // Role ID is required
	Role         										 Role           `gorm:"foreignKey:RoleID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"role,omitempty"`
	DOB          										 *time.Time     `json:"dob" gorm:"type:date;null"` // Date of birth, nullable
	Gender       										 string         `gorm:"type:varchar(255);null" json:"gender"` // Gender, nullable
	Token        										 string         `gorm:"type:text;null" json:"token"`                                // Token is optional and can be text
	ActiveStatus 										 bool           `gorm:"default:true" json:"active_status" validate:"required"`      // Active status is required (default is true)
	IsDeleted   										 bool           `gorm:"default:false" json:"is_deleted"`                            // Soft delete flag (default is false)
	CreatedAt   										 time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"` // Timestamp of creation
	UpdatedAt   										 time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"` // Timestamp of last update
	DeletedAt   										 gorm.DeletedAt `gorm:"index" json:"-"`                                             // Soft deletion, hidden in API responses
	Kits        										 []Kit          `gorm:"foreignKey:CreatedBy" json:"kits,omitempty"`                 // Relationship with Kits (optional, as user may not have kits)
	DefaultPatientSignatureURL			 string 				`gorm:"type:varchar(255);null" json:"default_patient_signature_url"`
	DefaultPatientSignatureHash			 string 				`gorm:"type:varchar(128);null" json:"default_patient_signature_hash"`
	DefaultNurseSignatureURL				 string 				`gorm:"type:varchar(255);null" json:"default_nurse_signature_url"`
	DefaultNurseSignatureHash				 string					`gorm:"type:varchar(128);null" json:"default_nurse_signature_hash"`
	NHI										           string					`gorm:"type:varchar(20);null" json:"nhi,omitempty" validate:"omitempty,max=20"`
}
