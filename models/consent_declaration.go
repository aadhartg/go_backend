package models

type ConsentDeclaration struct {
	ID              uint         `gorm:"primaryKey" json:"id"`
	EConsentFormID  uint         `gorm:"not null" json:"econsent_form_id"`
	EConsentForm    EConsentForm `gorm:"foreignKey:EConsentFormID" json:"econsent_form"`
	DeclarationID   uint         `gorm:"not null" json:"declaration_id"`
	Declaration     Declaration  `gorm:"foreignKey:DeclarationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"declaration"`
	OrderIndex      int          `gorm:"not null" json:"order_index"`
} 