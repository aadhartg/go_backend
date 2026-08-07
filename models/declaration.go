package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// StringArray is a custom type for []string to support JSONB in PostgreSQL
// Implements driver.Valuer and sql.Scanner
//
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *StringArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	return json.Unmarshal(bytes, a)
}

type ResponseType string

const (
	ResponseTypeYesNo     ResponseType = "yes_no"
	ResponseTypeText      ResponseType = "text"
	ResponseTypeMultiText ResponseType = "multi_text"
)

type Declaration struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	FormID        uint           `gorm:"not null" json:"form_id"`
	Statement     string         `gorm:"type:text;not null" json:"statement"`
	Title         string         `gorm:"type:varchar(255);not null" json:"title"`
	ResponseType  ResponseType   `gorm:"type:varchar(20);not null" json:"response_type"`
	Options       StringArray    `gorm:"type:jsonb" json:"options,omitempty"`
	CreatedBy     uint           `gorm:"not null" json:"created_by"`
	UpdatedBy     *uint          `gorm:"null" json:"updated_by"`
	User          User           `gorm:"foreignKey:CreatedBy" json:"user"`
	UpdatedByUser User           `gorm:"foreignKey:UpdatedBy" json:"updated_by_user,omitempty"`
	Form          DeclarationForm `gorm:"foreignKey:FormID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"form,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	IsDeleted     bool           `gorm:"default:false" json:"is_deleted"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
