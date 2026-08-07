package models

import (
	"time"

	"gorm.io/gorm"
)

type Video struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"type:varchar(255);unique;not null" json:"title" validate:"required"`
	Description string    `gorm:"type:text;not null" json:"description"`
	FilePath    string    `gorm:"not null;column:file_path" json:"file_path"`
	FileSize    int64     `gorm:"not null" json:"file_size"` // Size in bytes
	MimeType    string    `gorm:"not null" json:"mime_type"` // e.g., video/mp4
	Duration    float64   `gorm:"not null" json:"duration"`  // Duration in seconds
	UploadedBy  uint      `gorm:"not null" json:"uploaded_by"`
	User        User      `gorm:"foreignKey:UploadedBy;references:ID" json:"user,omitempty"` // Uploaded by user
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	IsDeleted   bool      `gorm:"default:false" json:"is_deleted"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}