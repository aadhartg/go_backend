package models

type EConsentStatus string

const (
	EConsentStatusAssigned   EConsentStatus = "assigned"
	EConsentStatusInProgress EConsentStatus = "in_progress"
	EConsentStatusSubmitted  EConsentStatus = "submitted"
	EConsentStatusApproved   EConsentStatus = "approved"
	EConsentStatusExpired    EConsentStatus = "expired"
	EConsentStatusWithdrawn  EConsentStatus = "withdrawn"
	EConsentStatusReviewed   EConsentStatus = "reviewed"
	EConsentStatusArchived   EConsentStatus = "archived"
)