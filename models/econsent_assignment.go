package models

import (
	"time"

	"gorm.io/gorm"
)

type EConsentAssignment struct {
	ID                    uint                `gorm:"primaryKey" json:"id"`
	EConsentFormID        uint                `gorm:"not null" json:"econsent_form_id"`
	EConsentForm          EConsentForm        `gorm:"foreignKey:EConsentFormID" json:"econsent_form"`
	EConsentFormVersionID uint                `gorm:"not null" json:"econsent_form_version_id"`                      // New: links to versioned snapshot
	EConsentFormVersion   EConsentFormVersion `gorm:"foreignKey:EConsentFormVersionID" json:"econsent_form_version"` // <-- Add this line
	PatientID             uint                `gorm:"not null" json:"patient_id"`
	User                  User                `gorm:"foreignKey:PatientID" json:"user"`
	AssignedAt            time.Time           `json:"assigned_at"`
	Status                EConsentStatus      `gorm:"type:varchar(20);not null" json:"status"`
	SubmittedAt           *time.Time          `json:"submitted_at,omitempty"`
	ApprovedAt            *time.Time          `json:"approved_at,omitempty"`
	IsDeleted             bool                `gorm:"default:false" json:"is_deleted"`
	DeletedAt             gorm.DeletedAt      `gorm:"index" json:"deleted_at,omitempty"`
	WithdrawalReason      string              `gorm:"type:text;null" json:"withdrawal_reason,omitempty"` // Reason for withdrawal
	WithdrawnReviewedBy   uint                `json:"withdrawn_reviewed_by"`
	WithdrawnReviewedAt   *time.Time          `json:"withdrawn_reviewed_at"`
	IsVerified            bool                `gorm:"default:false" json:"is_verified"`
}
