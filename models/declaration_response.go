package models

import (
	"time"

	"gorm.io/gorm"
)

type DeclarationResponse struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	PatientEConsentID uint           `gorm:"not null" json:"patient_econsent_id"`
	PatientEConsent   PatientEConsent `gorm:"foreignKey:PatientEConsentID" json:"patient_econsent"`
	DeclarationID     uint           `gorm:"not null" json:"declaration_id"`
	Declaration       Declaration    `gorm:"foreignKey:DeclarationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"declaration"`
	Response          string          `json:"response,omitempty"`
	ResponseText      string         `gorm:"type:text" json:"response_text,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	IsDeleted         bool           `gorm:"default:false" json:"is_deleted"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
} 