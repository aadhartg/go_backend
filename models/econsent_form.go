package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// PatientAssignedArray is a custom type to handle JSON serialization for patient IDs
type PatientAssignedArray []uint

// Value implements the driver.Valuer interface
func (pa PatientAssignedArray) Value() (driver.Value, error) {
	if pa == nil {
		return nil, nil
	}
	return json.Marshal(pa)
}

// Scan implements the sql.Scanner interface
func (pa *PatientAssignedArray) Scan(value interface{}) error {
	if value == nil {
		*pa = nil
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, pa)
	case string:
		return json.Unmarshal([]byte(v), pa)
	default:
		return nil
	}
}

type EConsentFormDeclarations struct {
	EConsentFormID    uint `gorm:"primaryKey"`
	DeclarationFormID uint `gorm:"primaryKey"`
	OrderIndex        int
}

type EConsentForm struct {
	ID                uint             `gorm:"primaryKey" json:"id"`
	ResearchID        uint             `gorm:"not null" json:"research_id"`
	Research          Research         `gorm:"foreignKey:ResearchID" json:"research"`
	CreatedBy         uint             `gorm:"not null" json:"created_by"`
	User              User             `gorm:"foreignKey:CreatedBy" json:"user"`
	Title             string           `gorm:"not null" json:"title"`
	// PisContent  string           `gorm:"type:text" json:"pis_content"`
	VideoID           uint             `gorm:"not null" json:"video_id"`
	VideoDescription  string           `gorm:"type:text" json:"video_description"`
	PersonalInfoCheck bool             `gorm:"default:false" json:"personal_info_check"`
	PatientAssigned   PatientAssignedArray `gorm:"type:json" json:"patient_assigned"` // Multiple patient IDs
	ExpiryTime        time.Time        `gorm:"not null" json:"expiry_time"`
	Video             Video            `gorm:"foreignKey:VideoID" json:"video"`
	IsPublished       bool             `gorm:"default:false" json:"is_published"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	IsDeleted         bool             `gorm:"default:false" json:"is_deleted"`
	DeletedAt         gorm.DeletedAt   `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	Declarations     []ConsentDeclaration `gorm:"foreignKey:EConsentFormID" json:"declarations,omitempty"`
	Assignments      []EConsentAssignment `gorm:"foreignKey:EConsentFormID" json:"assignments,omitempty"`
	PatientConsents  []PatientEConsent    `gorm:"foreignKey:EConsentFormID" json:"patient_consents,omitempty"`
	CoordinatorIDs   PatientAssignedArray `gorm:"type:json" json:"coordinator_ids"` // Multiple coordinator IDs
	DeclarationForms []DeclarationForm `gorm:"many2many:econsent_form_declaration_forms;"`
} 