package models

import (
	"time"
)

type DeclarationForm struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null;unique" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	CreatedBy   uint      `gorm:"not null" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	Declarations []Declaration `gorm:"foreignKey:FormID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"declarations"`
} 