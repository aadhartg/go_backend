package models

import (
	"time"

	"gorm.io/gorm"
)

type ResearchCategory struct {
	ID            uint   							`gorm:"primaryKey;autoIncrement;not null" json:"id"`
	Name          string 							`gorm:"type:varchar(40);not null;" json:"name"`
	CreatedBy     uint        				`gorm:"not null" json:"created_by" validate:"required"`        
	CreatedByUser User  							`gorm:"foreignKey:CreatedBy;references:ID" json:"created_by_user,omitempty"`
	CreatedAt 		time.Time 					`gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt 		time.Time 					`gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsDeleted 		bool      					`gorm:"default:false" json:"is_deleted"`
	DeletedAt 		gorm.DeletedAt 			`gorm:"index" json:"-"`
}