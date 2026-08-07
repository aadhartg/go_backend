// models/role.go

package models

import (
	"time"

	"gorm.io/gorm"
)

// Role model
type Role struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"type:varchar(20);unique;not null" json:"name" validate:"required,max=20"` // Limit to 20 characters and required
	Status    bool           `gorm:"default:true" json:"status"`
	IsDeleted bool           `gorm:"default:false" json:"is_deleted"`
	CreatedAt time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	Users     []User         `gorm:"foreignKey:RoleID" json:"users,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
