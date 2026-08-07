package models

import (
	"time"
)

// StudyResult represents a document uploaded for a patient's participation in a study
// Only nurses can publish these results
// File is stored in /uploads/results/
type Genomics struct {
	ID                    uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	PatientID             uint         `gorm:"not null" json:"patient_id"`
	Patient               User         `gorm:"foreignKey:PatientID"`
	ResearchID            uint         `gorm:"not null" json:"research_id"`
	Research              Research     `gorm:"foreignKey:ResearchID"`
	EConsentForm          EConsentForm `gorm:"foreignKey:EConsentFormID"`
	EConsentFormID        uint         `gorm:"not null" json:"e_consent_form_id"`
	EConsentFormVersionID *uint        `gorm:"index" json:"e_consent_form_version_id,omitempty"` // New: links to specific consent form version
	EConsentFormVersion   *EConsentFormVersion `gorm:"foreignKey:EConsentFormVersionID" json:"e_consent_form_version,omitempty"`
	FileURL               string       `gorm:"type:varchar(255);not null" json:"file_url"`
	FileHash              string       `gorm:"type:varchar(64);not null" json:"file_hash"`
	PublishedBy           uint         `gorm:"not null" json:"published_by"` // Nurse user ID
	IsPublished           bool         `gorm:"default:false" json:"is_published"` // Publication status
	CreatedAt             time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time    `gorm:"autoUpdateTime" json:"updated_at"`
	IsDeleted             bool         `gorm:"default:false" json:"is_deleted"`
}