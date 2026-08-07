package models

import (
	"time"

	"gorm.io/gorm"
)

type PatientEConsent struct {
	ID                    uint                `gorm:"primaryKey" json:"id"`
	PatientID             uint                `gorm:"not null" json:"patient_id"`
	Patient               User                `gorm:"foreignKey:PatientID" json:"patient"`
	EConsentFormID        uint                `gorm:"not null" json:"econsent_form_id"`
	EConsentForm          EConsentForm        `gorm:"foreignKey:EConsentFormID" json:"econsent_form"`
	EConsentFormVersionID uint                `json:"econsent_form_version_id"`
	EConsentFormVersion   EConsentFormVersion `gorm:"foreignKey:EConsentFormVersionID" json:"econsent_form_version"`
	Status                EConsentStatus      `gorm:"type:varchar(20);not null" json:"status"`
	NHI                   string              `gorm:"type:varchar(255);not null" json:"nhi"`
	FirstName             string              `gorm:"type:varchar(255);not null" json:"first_name"`
	LastName              string              `gorm:"type:varchar(255);not null" json:"last_name"`
	Gender                string              `gorm:"type:varchar(255);not null" json:"gender"`
	DOB                   string              `gorm:"type:varchar(255);not null" json:"dob"`
	PatientSignURL        string              `gorm:"type:varchar(255)" json:"patient_sign_url"`
	PatientSignedAt       *time.Time          `json:"patient_signed_at"`
	NurseID               uint                `json:"nurse_id"`
	Nurse                 User                `gorm:"foreignKey:NurseID" json:"nurse"`
	NurseSignURL          string              `gorm:"type:varchar(255)" json:"nurse_sign_url"`
	NurseSignedAt         *time.Time          `json:"nurse_signed_at"`
	PatientSignHash       string              `gorm:"type:varchar(128);null" json:"patient_sign_hash"`
	NurseSignHash         string              `gorm:"type:varchar(128);null" json:"nurse_sign_hash"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
	IsDeleted             bool                `gorm:"default:false" json:"is_deleted"`
	DeletedAt             gorm.DeletedAt      `gorm:"index" json:"deleted_at,omitempty"`
	WithdrawalReason      string              `gorm:"type:text;null" json:"withdrawal_reason,omitempty"` // Reason for withdrawal
	WithdrawnReviewedBy   uint                `json:"withdrawn_reviewed_by"`
	WithdrawnReviewedAt   *time.Time          `json:"withdrawn_reviewed_at"`
	SubmittedBy           uint                `json:"submitted_by"`
	SubmittedByUser       User                `gorm:"foreignKey:SubmittedBy" json:"submitted_by_user"`
}
