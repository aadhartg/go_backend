package models

import "time"

type EConsentFormVersion struct {
	ID             uint      `gorm:"primaryKey"`
	EConsentFormID uint      `gorm:"not null;index"`
	VersionNumber  int       `gorm:"not null"`
	SnapshotJSON   string    `gorm:"type:text;not null"`
	PublishedAt    time.Time `gorm:"not null"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
