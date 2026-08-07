package models

// models/consent_session.go

import "time"

type ConsentSession struct {
	ID    uint   `gorm:"primaryKey"`
	Token string `gorm:"type:varchar(255);uniqueIndex;not null"`

	AssignmentID   uint `gorm:"not null"`
	PatientID      uint `gorm:"not null"`
	NurseID        uint `gorm:"not null"`
	EConsentFormID uint `gorm:"not null"`

	IsActive  bool      `gorm:"default:true"`
	ExpiresAt time.Time `gorm:"not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
