package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	// "regexp"
	"strconv"
	"strings"
	"time"

	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeclarationResponseInput struct {
	DeclarationID uint        `json:"declaration_id" binding:"required"`
	Response      interface{} `json:"response"`
}

type FormDeclarationInput struct {
	FormID      uint
	Declaration []DeclarationResponseInput
}

func CreatePatientEConsentService(
	tx *gorm.DB,
	user *models.User,
	patientID uint,
	submittedBy uint,
	eConsentFormID uint,
	eConsentFormVersionID uint,
	nhi, firstName, lastName, gender string,
	dob string,
	declarationResponses []FormDeclarationInput, // encrypted JSON string
	c *gin.Context, // for file upload
) error {

	parsedDOB, err := time.Parse("2006-01-02", dob)
	if err != nil {
		tx.Rollback()
		return errors.New("Invalid DOB format, use YYYY-MM-DD")
	}

	// Prevent duplicate submissions
	var existingConsentForForm models.PatientEConsent
	if err := tx.Where("patient_id = ? AND e_consent_form_id = ? AND e_consent_form_version_id = ? AND is_deleted = ?", patientID, eConsentFormID, eConsentFormVersionID, false).First(&existingConsentForForm).Error; err == nil {
		return errors.New("You have already submitted consent for this version of the research.")
	}

	// Check if assignment exists
	var assignment models.EConsentAssignment
	if err := tx.Where("e_consent_form_id = ? AND patient_id = ? AND e_consent_form_version_id = ? AND is_deleted = ?", eConsentFormID, patientID, eConsentFormVersionID, false).First(&assignment).Error; err != nil {
		return errors.New("No e-consent assignment found for this patient")
	}

	// Ensure this assignment is the active one for this patient-form combination
	// Archive any other assignments for the same patient and form
	if err := tx.Model(&models.EConsentAssignment{}).
		Where("e_consent_form_id = ? AND patient_id = ? AND id != ? AND is_deleted = ?",
			eConsentFormID, patientID, assignment.ID, false).
		Update("status", models.EConsentStatusArchived).Error; err != nil {
		return errors.New("Failed to archive other assignments")
	}

	// Handle signature
	var patientSignURL, patientSignHash string
	if file, err := c.FormFile("patient_sign_file"); err == nil && file != nil {
		hash, fileBytes, err := utils.ComputeFileHashFromFormFile(file)
		if err != nil {
			return errors.New("Failed to compute signature hash")
		}
		signDir := "public/signatures/patients"
		_ = os.MkdirAll(signDir, 0755)
		fileName := fmt.Sprintf("patient_%d_%s%s", patientID, hash, filepath.Ext(file.Filename))
		filePath := filepath.Join(signDir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := os.WriteFile(filePath, fileBytes, 0644); err != nil {
				return errors.New("Failed to save new signature file")
			}
		}
		patientSignURL = "/signatures/patients/" + fileName
		patientSignHash = hash
		user.DefaultPatientSignatureURL = patientSignURL
		user.DefaultPatientSignatureHash = patientSignHash
		_ = tx.Save(user)
	} else if user.DefaultPatientSignatureURL != "" && user.DefaultPatientSignatureHash != "" {
		patientSignURL = user.DefaultPatientSignatureURL
		patientSignHash = user.DefaultPatientSignatureHash
	} else {
		return errors.New("A signature is required for the first submission.")
	}

	// Fetch the EConsentForm
	var eConsentForm models.EConsentForm
	if err := tx.Where("id = ?", eConsentFormID).First(&eConsentForm).Error; err != nil {
		return errors.New("E-consent form not found")
	}

	// Check expiry
	expired, msg := CheckEConsentExpiry(tx, eConsentFormID, nil, &assignment.ID)
	if expired {
		return errors.New(msg)
	}

	// Get nurse ID
	var nurseID uint
	if len(eConsentForm.CoordinatorIDs) > 0 {
		nurseID = eConsentForm.CoordinatorIDs[0]
	} else {
		return errors.New("No coordinator assigned to this e-consent form")
	}

	encryptedFirstName, err := utils.Encrypt(firstName)
	if err != nil {
		tx.Rollback()
		return errors.New("Failed to encrypt first name")
	}
	encryptedLastName, err := utils.Encrypt(lastName)
	if err != nil {
		tx.Rollback()
		return errors.New("Failed to encrypt last name")
	}
	encryptedGender, err := utils.Encrypt(gender)
	if err != nil {
		tx.Rollback()
		return errors.New("Failed to encrypt gender")
	}
	encryptedDOB, err := utils.Encrypt(parsedDOB.Format("2006-01-02"))
	if err != nil {
		tx.Rollback()
		return errors.New("Failed to encrypt DOB")
	}
	encryptedNHI, err := utils.Encrypt(nhi)
	if err != nil {
		tx.Rollback()
		return errors.New("Failed to encrypt NHI")
	}

	// Save decrypted NHI before overwriting with encrypted version
	decryptedNHIForUser := nhi

	firstName = encryptedFirstName
	lastName = encryptedLastName
	gender = encryptedGender
	dob = encryptedDOB
	nhi = encryptedNHI

	// Create PatientEConsent entry
	pe := models.PatientEConsent{
		PatientID:             patientID,
		SubmittedBy:           submittedBy,
		EConsentFormID:        eConsentFormID,
		EConsentFormVersionID: eConsentFormVersionID,
		Status:                models.EConsentStatusSubmitted,
		NHI:                   nhi,
		FirstName:             firstName,
		LastName:              lastName,
		Gender:                gender,
		DOB:                   dob, // Store as string, encrypt if needed
		PatientSignURL:        patientSignURL,
		PatientSignHash:       patientSignHash,
		PatientSignedAt:       utils.TimePtr(time.Now()),
		NurseID:               nurseID,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	if err := tx.Create(&pe).Error; err != nil {
		return errors.New("Failed to create patient e-consent entry")
	}

	// Save declaration responses for each form
	for _, formResp := range declarationResponses {
		for _, dr := range formResp.Declaration {
			// Instead of fetching from current DB, use the version snapshot
			// This ensures we work with the exact declarations that existed when the form was published
			var declaration models.Declaration

			// Try to find the declaration in the current DB first (for backward compatibility)
			if err := tx.Where("id = ?", dr.DeclarationID).First(&declaration).Error; err != nil {
				// If not found in current DB, try to get it from the version snapshot
				if eConsentFormVersionID > 0 {
					// Fetch the version snapshot
					var version models.EConsentFormVersion
					if err := tx.Where("id = ?", eConsentFormVersionID).First(&version).Error; err != nil {
						tx.Rollback()
						return errors.New("Version snapshot not found")
					}

					// Parse the snapshot to find the declaration
					var snapshot map[string]interface{}
					if err := json.Unmarshal([]byte(version.SnapshotJSON), &snapshot); err != nil {
						tx.Rollback()
						return errors.New("Failed to parse version snapshot")
					}

					// Look for the declaration in the snapshot
					declarationFound := false
					if declarationForms, ok := snapshot["declaration_forms"].([]interface{}); ok {
						for _, form := range declarationForms {
							if formMap, ok := form.(map[string]interface{}); ok {
								if statements, ok := formMap["statements"].([]interface{}); ok {
									for _, stmt := range statements {
										if stmtMap, ok := stmt.(map[string]interface{}); ok {
											if stmtID, ok := stmtMap["id"].(float64); ok && uint(stmtID) == dr.DeclarationID {
												// Found the declaration in snapshot
												declarationFound = true

												// Recreate the declaration from snapshot data to satisfy foreign key
												declaration.ID = dr.DeclarationID
												if title, ok := stmtMap["title"].(string); ok {
													declaration.Title = title
												}
												if statement, ok := stmtMap["statement"].(string); ok {
													declaration.Statement = statement
												}
												if responseType, ok := stmtMap["response_type"].(string); ok {
													declaration.ResponseType = models.ResponseType(responseType)
												} else {
													declaration.ResponseType = models.ResponseTypeText // default fallback
												}
												if options, ok := stmtMap["options"].([]interface{}); ok {
													var optionStrings []string
													for _, opt := range options {
														if optStr, ok := opt.(string); ok {
															optionStrings = append(optionStrings, optStr)
														}
													}
													declaration.Options = optionStrings
												}

												// Extract required fields from snapshot
												if formID, ok := stmtMap["form_id"].(float64); ok {
													declaration.FormID = uint(formID)
												} else {
													// Try to get form_id from the form level
													if formID, ok := formMap["id"].(float64); ok {
														declaration.FormID = uint(formID)
													} else {
														// Default to a reasonable form ID (1 is usually safe)
														declaration.FormID = uint(1)
													}
												}

												// Set created_by to the current user (or a default system user)
												declaration.CreatedBy = uint(1) // Default system user ID

												declaration.CreatedAt = time.Now()
												declaration.UpdatedAt = time.Now()

												// Temporarily insert the declaration to satisfy foreign key constraint
												// Use INSERT ... ON CONFLICT DO NOTHING to avoid duplicate key errors

												// Properly serialize options to JSON format
												var optionsJSON []byte
												var optionsErr error
												if len(declaration.Options) > 0 {
													optionsJSON, optionsErr = json.Marshal(declaration.Options)
													if optionsErr != nil {
														tx.Rollback()
														return errors.New("Failed to serialize declaration options")
													}
												} else {
													optionsJSON = []byte("[]") // Empty array as JSON
												}

												err := tx.Exec("INSERT INTO declarations (id, form_id, title, statement, response_type, options, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO NOTHING",
													declaration.ID, declaration.FormID, declaration.Title, declaration.Statement, declaration.ResponseType,
													optionsJSON, declaration.CreatedBy, declaration.CreatedAt, declaration.UpdatedAt).Error
												if err != nil {
													// If INSERT fails, try to fetch it again (might have been created by another transaction)
													if err := tx.Where("id = ?", dr.DeclarationID).First(&declaration).Error; err != nil {
														tx.Rollback()
														return errors.New("Failed to restore declaration from snapshot")
													}
												}
												break
											}
										}
									}
								}
							}
							if declarationFound {
								break
							}
						}
					}

					if !declarationFound {
						tx.Rollback()
						return errors.New("Declaration not found in version snapshot")
					}
				} else {
					// No version ID, fall back to current DB
					tx.Rollback()
					return errors.New("Declaration not found")
				}
			}

			var toSave string
			if declaration.ResponseType == "yes_no" {
				// Always save as "true" or "false"
				toSave = fmt.Sprintf("%v", dr.Response)
			} else {
				// Save as string (text, multi_text, etc.)
				toSave = fmt.Sprintf("%v", dr.Response)
			}
			encryptedResp, err := utils.Encrypt(toSave)
			if err != nil {
				tx.Rollback()
				return errors.New("Failed to encrypt declaration response")
			}
			declResp := models.DeclarationResponse{
				PatientEConsentID: pe.ID,
				DeclarationID:     dr.DeclarationID,
				Response:          encryptedResp, // store encrypted string
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			}
			if err := tx.Create(&declResp).Error; err != nil {
				tx.Rollback()
				return errors.New("Failed to save declaration responses")
			}
		}
	}

	// Update assignment status
	assignment.Status = models.EConsentStatusSubmitted
	assignment.SubmittedAt = utils.TimePtr(time.Now())
	if err := tx.Save(&assignment).Error; err != nil {
		return errors.New("Failed to update assignment status")
	}

	// Also update any other assignments for this patient and form to ensure consistency
	// This prevents multiple active assignments for the same patient-form combination
	if err := tx.Model(&models.EConsentAssignment{}).
		Where("patient_id = ? AND e_consent_form_id = ? AND id != ? AND is_deleted = ?",
			patientID, eConsentFormID, assignment.ID, false).
		Update("status", models.EConsentStatusArchived).Error; err != nil {
		return errors.New("Failed to archive other assignments")
	}

	// Update user's DOB, Gender, and NHI if provided
	user.DOB = &parsedDOB
	user.Gender = gender
	// Update NHI in user model only if it doesn't exist (first time only)
	// Use decryptedNHIForUser (saved before encryption) to store plain text in user model
	// This makes user.NHI the source of truth and prevents it from being overwritten during versioning
	if user.NHI == "" && decryptedNHIForUser != "" {
		user.NHI = decryptedNHIForUser
	}
	if err := tx.Save(user).Error; err != nil {
		return fmt.Errorf("Failed to update user profile with DOB, gender, and NHI: %w", err)
	}

	// Send notifications to ALL coordinators associated with this e-consent form
	// This ensures all coordinators are notified when a patient submits consent
	if len(eConsentForm.CoordinatorIDs) > 0 {
		// Fetch coordinator details for notifications
		var coordinators []models.User

		// Handle the case where CoordinatorIDs might be stored as a string representation
		var coordinatorIDs []uint
		switch v := any(eConsentForm.CoordinatorIDs).(type) {
		case []uint:
			coordinatorIDs = v
		case string:
			// Try to unmarshal if it's a string
			if err := json.Unmarshal([]byte(v), &coordinatorIDs); err != nil {
				fmt.Printf("WARNING: Failed to unmarshal coordinator IDs from string: %v\n", err)
				// Try to extract numbers from the string manually as a fallback
				coordinatorIDs = extractNumbersFromString(v)
			}
		case models.PatientAssignedArray:
			coordinatorIDs = []uint(v)
		default:
			fmt.Printf("WARNING: Unknown type for CoordinatorIDs: %T\n", v)
			coordinatorIDs = []uint{}
		}

		if len(coordinatorIDs) > 0 {
			// Use a different approach for the IN clause to avoid PostgreSQL parameter binding issues
			query := tx.Where("id IN (?)", coordinatorIDs)
			if err := query.Find(&coordinators).Error; err != nil {
				// Log error but don't fail the main operation
				fmt.Printf("WARNING: Failed to fetch coordinators for notifications: %v\n", err)
			} else {
				// Send notification to each coordinator
				for _, coordinator := range coordinators {
					title := "Patient Consent Submitted"
					message := fmt.Sprintf("Patient %s %s has submitted consent for e-consent form '%s'. Status: Pending from Coordinator. Please review and take action.",
						user.FirstName, user.LastName, eConsentForm.Title)
					notificationType := "E-Consent-Review"
					entityType := "econsents"
					entityID := eConsentFormID
					username := user.FirstName + " " + user.LastName
					metadata := map[string]interface{}{
						"econsent_form_id": eConsentFormID,
						"patient_id":       patientID,
						"status":           "pending_from_coordinator",
						"action_required":  "review_patient_consent",
					}

					// Create notification for this coordinator
					if err := CreateNotification(tx, coordinator.ID, title, message, notificationType, username, entityType, entityID, metadata); err != nil {
						// Log error but don't fail the main operation
						fmt.Printf("WARNING: Failed to create notification for coordinator %d: %v\n", coordinator.ID, err)
					}
				}
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return errors.New("Failed to commit transaction")
	}

	return nil
}

// extractNumbersFromString extracts numeric IDs from a string representation like "[12]" or "12,13,14"
func extractNumbersFromString(s string) []uint {
	var result []uint

	// Remove brackets and split by comma
	cleanStr := strings.Trim(s, "[]")
	if cleanStr == "" {
		return result
	}
	// Split by comma and convert each part to uint
	parts := strings.Split(cleanStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			if num, err := strconv.ParseUint(part, 10, 32); err == nil {
				result = append(result, uint(num))
			}
		}
	}
	return result
}
