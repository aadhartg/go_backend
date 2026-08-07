// models/notification.go
package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Notification model
type Notification struct {
	ID         uint            `gorm:"primaryKey;autoIncrement;not null" json:"id"`
	UserID     uint            `gorm:"not null" json:"user_id" validate:"required"` // Recipient of the notification
	User       User            `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	Title      string          `gorm:"type:varchar(255);not null" json:"title" validate:"required"`
	Message    string          `gorm:"type:text;not null" json:"message" validate:"required"`
	Username   string          `gorm:"type:text;" json:"username" validate:"required"`
	Type       string          `gorm:"type:varchar(50);not null" json:"dna_notification_type" validate:"required,oneof=Admin Management Inventory Management Order Purchase Order Management Genomic-Results"`
	EntityID   uint            `gorm:"not null" json:"entity_id"` // ID of the related entity (e.g., user, kit, order)
	EntityType string          `gorm:"type:varchar(50);not null" json:"dna_notification_entity_type" validate:"required,oneof=barcodes customers invoices kits orders payments roles users genomics"`
	Metadata   json.RawMessage `gorm:"type:jsonb;default:'{}'" json:"metadata"` // New flexible JSON field
	IsRead     bool            `gorm:"default:false" json:"is_read"`
	CreatedAt  time.Time       `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time       `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsDeleted  bool            `gorm:"default:false" json:"is_deleted"`
	DeletedAt  gorm.DeletedAt  `gorm:"index" json:"-"`
}
