// services/notification_service.go
package services

import (
	"theransticslabs/m/models"
	"time"

	"gorm.io/gorm"
)

// CheckEConsentExpiry checks if the e-consent form is expired and updates assignment/patient e-consent status if needed.
// Returns (true, message) if expired, (false, "") if not expired.
func CheckEConsentExpiry(db *gorm.DB, eConsentFormID uint, patientEConsentID *uint, assignmentID *uint) (bool, string) {
	var eConsentForm models.EConsentForm
	if err := db.Where("id = ?", eConsentFormID).First(&eConsentForm).Error; err != nil {
		return true, "E-consent form not found."
	}
	// If checking for PatientEConsent
	if patientEConsentID != nil {
		var pe models.PatientEConsent
		if err := db.First(&pe, *patientEConsentID).Error; err == nil {
			if pe.Status == models.EConsentStatusApproved {
				// Already approved, do not expire
				return false, ""
			}
			// Only expire if not submitted or in progress
			if pe.Status == models.EConsentStatusSubmitted {
				// Already submitted, do not expire
				return false, ""
			}
		}
	}
	// If checking for Assignment
	if assignmentID != nil {
		var assignment models.EConsentAssignment
		if err := db.First(&assignment, *assignmentID).Error; err == nil {
			if assignment.Status == models.EConsentStatusApproved {
				// Already approved, do not expire
				return false, ""
			}
			// Only expire if not submitted or in progress
			if assignment.Status == models.EConsentStatusSubmitted {
				// Already submitted, do not expire
				return false, ""
			}
		}
	}
	if time.Now().After(eConsentForm.ExpiryTime) {
		var notifiedPatientID uint
		var notifyPatient bool
		if assignmentID != nil {
			var assignment models.EConsentAssignment
			if err := db.First(&assignment, *assignmentID).Error; err == nil {
				if assignment.Status != models.EConsentStatusExpired {
					db.Model(&models.EConsentAssignment{}).Where("id = ?", *assignmentID).Update("status", models.EConsentStatusExpired)
					notifiedPatientID = assignment.PatientID
					notifyPatient = true
				}
			}
		}
		if patientEConsentID != nil {
			var pe models.PatientEConsent
			if err := db.First(&pe, *patientEConsentID).Error; err == nil {
				if pe.Status != models.EConsentStatusExpired {
					db.Model(&models.PatientEConsent{}).Where("id = ?", *patientEConsentID).Update("status", models.EConsentStatusExpired)
					notifiedPatientID = pe.PatientID
					notifyPatient = true
				}
			}
		}
		// Notify patient if needed
		if notifyPatient && notifiedPatientID != 0 {
			CreateNotification(
				db,
				notifiedPatientID,
				"Consent Form Expired",
				"Your assigned consent form has expired. Please request a new one.",
				"E-Consent",
				"",
				"econsents",
				eConsentFormID,
				map[string]interface{}{"status": "expired"},
			)
		}
		// Notify all coordinators/nurses
		for _, nurseID := range eConsentForm.CoordinatorIDs {
			CreateNotification(
				db,
				nurseID,
				"Consent Form Expired",
				"A patient's assigned consent form has expired.",
				"E-Consent",
				"",
				"econsents",
				eConsentFormID,
				map[string]interface{}{"status": "expired"},
			)
		}
		return true, "This consent form has expired. Please request a new one."
	}
	return false, ""
}
