package models

import "time"

type PublishedResearch struct {
	ID          			uint 			`gorm:"primaryKey;autoIncrement;not null;" json:"id"`
	ResearchID  			uint 			`gorm:"not null" json:"research_id" validate:"required"`
	PatientID   			uint 			`gorm:"not null" json:"patient_id" validate:"required"`
	PublishedAt 			time.Time `gorm:"autoUpdateTime;column:published_at" json:"updated_at"`
	PublishedBy 			uint  		`gorm:"not null" json:"published_by"`
	PublishedByUser  	User 			`gorm:"foreignKey:PublishedBy;references:ID" json:"published_by_user,omitempty"`
}