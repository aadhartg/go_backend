// models/errorLogs.go

package models

import (
	"time"

	"gorm.io/gorm"
)

type ErrorLog struct {
	ID              uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	APIPath         string         `gorm:"size:255;not null"`
	ErrorMessage    string         `gorm:"type:text"`
	ErrorType       string         `gorm:"size:50;not null"` // "payment" or "general"
	OccurrenceCount int            `gorm:"default:1"`
	LastOccurredAt  time.Time      `gorm:"autoUpdateTime"`
	CreatedAt       time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsDeleted       bool           `gorm:"default:false" json:"is_deleted"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}
