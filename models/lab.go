// models/lab.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// Lab represents the "labs" table in the database.
type Lab struct {
	ID         uint           `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	LabName    string         `gorm:"type:varchar(255);not null;column:lab_name" json:"lab_name" validate:"required,max=255"`
	LabAddress string         `gorm:"type:text;not null;column:lab_address" json:"lab_address" validate:"required"`
	NHINumber  string         `gorm:"type:varchar(10);not null;column:nhi_number" json:"nhi_number" validate:"required,max=10"`
	IsDeleted  bool           `gorm:"default:false;column:is_deleted" json:"is_deleted"`  // Soft delete flag
	CreatedAt  time.Time      `gorm:"autoCreateTime;column:created_at" json:"created_at"` // Creation timestamp
	UpdatedAt  time.Time      `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"` // Update timestamp
	DeletedAt  gorm.DeletedAt `gorm:"index;column:deleted_at" json:"-"`                   // Soft deletion timestamp
}
