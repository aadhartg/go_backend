package models

import (
	"time"

	"gorm.io/gorm"
)

type Research struct {
	ID            uint           `gorm:"primaryKey;autoIncrement;not null;" json:"id"`
	Title         string         `gorm:"type:varchar(30); not null;" json:"title" valiate:"required"`
	Description   string         `gorm:"type:varchar(100);not null;" json:"description" validate:"required"`
	CategoryID    uint           `gorm:"not null" json:"category_id" validate:"required"`
	CreatedBy     uint           `gorm:"not null" json:"created_by" validate:"required"`
	CreatedByUser User           `gorm:"foreignKey:CreatedBy;references:ID" json:"created_by_user,omitempty"`
	CreatedAt     time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsDeleted     bool           `gorm:"default:false" json:"is_deleted"`
	IsPublished   bool           `gorm:"default:false" json:"is_published"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Add this association for related documents
	Documents     []ResearchDocument `gorm:"foreignKey:ResearchID" json:"documents,omitempty"`
}
