package models

import (
	"time"

	"gorm.io/gorm"
)

type ResearchDocument struct {
	ID             uint           `gorm:"primaryKey;autoIncrement;not null" json:"id"`
	ResearchID     uint           `gorm:"not null" json:"research_id" validate:"required"`
	FilePath       string         `gorm:"type:varchar(255);column:file_path" json:"file_path"`
	FileType       string         `gorm:"type:varchar(255)" json:"file_type"`
	FileHash       string         `gorm:"type:varchar(64);column:file_hash" json:"file_hash"` // SHA-256 hash of the file
	UploadedBy     uint           `gorm:"not null" json:"uploaded_by"`
	UploadedByUser User           `gorm:"foreignKey:UploadedBy;references:ID" json:"uploaded_by_user,omitempty"`
	CreatedAt      time.Time      `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"` // Update timestamp
	IsDeleted      bool           `gorm:"default:false;column:is_deleted" json:"is_deleted"`  // Soft delete flag
	DeletedAt      gorm.DeletedAt `gorm:"index;column:deleted_at" json:"-"`                   // Soft deletion timestamp
}
