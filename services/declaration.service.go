package services

import (
	"errors"
	"time"

	"theransticslabs/m/models"

	"gorm.io/gorm"
)

type CreateDeclarationInput struct {
	FormID       uint
	Statement    string
	Title        string
	ResponseType models.ResponseType
	Options      models.StringArray
	CreatedBy    uint
}

func CreateDeclaration(tx *gorm.DB, input CreateDeclarationInput) (*models.Declaration, error) {
	// Check for duplicate title in DB (within the form)
	var count int64
	tx.Model(&models.Declaration{}).Where("form_id = ? AND title = ?", input.FormID, input.Title).Count(&count)
	if count > 0 {
		return nil, errors.New("The title '" + input.Title + "' already exists. Please enter a unique declaration title.",)
	}

	// Validate response_type and options
	if input.ResponseType == models.ResponseTypeText {
		if len(input.Options) == 0 {
			return nil, errors.New("Options are required for text response_type: " + input.Title)
		}
	} else {
		input.Options = models.StringArray{}
	}

	declaration := &models.Declaration{
		FormID:       input.FormID,
		Statement:    input.Statement,
		Title:        input.Title,
		ResponseType: input.ResponseType,
		Options:      input.Options,
		CreatedBy:    input.CreatedBy,
		CreatedAt:    time.Now(),
	}

	if err := tx.Create(declaration).Error; err != nil {
		return nil, err
	}
	return declaration, nil
}
