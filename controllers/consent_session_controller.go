package controllers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateConsentSession(c *gin.Context) {
	user, ok := middlewares.GetUserFromContext(c)

	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var req struct {
		AssignmentID uint `json:"assignmentId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	var assignment models.EConsentAssignment

	err := config.DB.
		Where("id = ?", req.AssignmentID).
		First(&assignment).Error

	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "Assignment not found", nil)
		return
	}

	token := uuid.NewString()

	session := models.ConsentSession{
		Token:          token,
		AssignmentID:   assignment.ID,
		PatientID:      assignment.PatientID,
		EConsentFormID: assignment.EConsentFormID,
		NurseID:        user.ID,
		IsActive:       true,
		ExpiresAt:      time.Now().Add(2 * time.Hour),
	}

	config.DB.Create(&session)

	utils.JSONResponse(c, http.StatusOK, "Session created", gin.H{
		"token": token,
	})
}

func GetConsentSessionFormDetails(c *gin.Context) {
	session, ok := middlewares.GetConsentSession(c)

	if !ok {
		utils.JSONResponse(
			c,
			http.StatusUnauthorized,
			"Invalid consent session",
			nil,
		)
		return
	}

	patientId := session.PatientID
	assignmentID := session.AssignmentID
	formID := session.EConsentFormID

	var version models.EConsentFormVersion
	err := config.DB.Where("e_consent_form_id = ?", formID).
		Order("version_number DESC").
		First(&version).Error
	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "E-consent version not found", nil)
		return
	}

	// Unmarshal the snapshot JSON
	var snapshot struct {
		Research         map[string]interface{}   `json:"research"`
		Video            map[string]interface{}   `json:"video"`
		VideoDescription string                   `json:"video_description"`
		DeclarationForms []map[string]interface{} `json:"declaration_forms"`
	}
	if err := json.Unmarshal([]byte(version.SnapshotJSON), &snapshot); err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to parse e-consent version snapshot", nil)
		return
	}

	// Fetch the EConsentForm for non-versioned fields
	var form models.EConsentForm
	err = config.DB.Where("id = ? AND is_deleted = ?", uint(formID), false).First(&form).Error
	if err != nil {
		utils.JSONResponse(c, http.StatusNotFound, "E-consent form not found", nil)
		return
	}

	// Try to find an existing PatientEConsent for this patient and form to prefill NHI and DOB
	var patientEConsent models.PatientEConsent
	err = config.DB.Where("patient_id = ? AND e_consent_form_id = ? AND e_consent_form_version_id = ? AND is_deleted = ?", patientId, formID, version.ID, false).Order("created_at desc").First(&patientEConsent).Error

	if err == nil {

		utils.JSONResponse(
			c,
			http.StatusOK,
			"E-consent already submitted",
			map[string]interface{}{
				"alreadySubmitted": true,
			},
		)

		return
	}

	var firstName, lastName, gender, nhi, dob string
	if err == nil {
		firstName = patientEConsent.FirstName
		lastName = patientEConsent.LastName
		gender = patientEConsent.Gender
		nhi = patientEConsent.NHI
		dob = patientEConsent.DOB
	} else if form.PersonalInfoCheck {
		// Prefill from user model if personal_info_check is true and no PatientEConsent exists
		var patient models.User

		err := config.DB.
			Where("id = ?", patientId).
			First(&patient).Error

		if err != nil {
			utils.JSONResponse(
				c,
				http.StatusNotFound,
				"Patient not found",
				nil,
			)
			return
		}

		encFirstName, _ := utils.Encrypt(patient.FirstName)
		encLastName, _ := utils.Encrypt(patient.LastName)
		encNHI, _ := utils.Encrypt(patient.NHI)

		encDOB := ""

		if patient.DOB != nil {
			encDOB, _ = utils.Encrypt(
				patient.DOB.Format("2006-01-02"),
			)
		}
		encGender := patient.Gender
		firstName = encFirstName
		lastName = encLastName
		gender = encGender
		dob = encDOB
		nhi = encNHI
	}

	// Prepare research document info (return the first document if exists)
	var researchDoc map[string]interface{}
	if err := config.DB.Preload("Documents").First(&form.Research, form.ResearchID).Error; err == nil {
		if len(form.Research.Documents) > 0 {
			doc := form.Research.Documents[0]
			researchDoc = map[string]interface{}{
				"id":        doc.ID,
				"file_name": filepath.Base(doc.FilePath),
				"file_type": doc.FileType,
			}
		}
	}

	// Prepare video info for response (from versioned snapshot if available, else from live)
	var videoInfo map[string]interface{}
	if snapshot.Video != nil && len(snapshot.Video) > 0 {
		videoInfo = snapshot.Video
	} else if form.Video.ID != 0 {
		videoInfo = map[string]interface{}{
			"id":          form.Video.ID,
			"title":       form.Video.Title,
			"description": form.Video.Description,
			"file_path":   filepath.Base(form.Video.FilePath),
		}
	}

	// Transform declaration_forms to required format
	var formattedDeclarationForms []map[string]interface{}
	for _, form := range snapshot.DeclarationForms {
		statements := []map[string]interface{}{}
		// Try both "declarations" and "statements" keys for compatibility
		var decls []interface{}
		if d, ok := form["declarations"]; ok {
			decls, _ = d.([]interface{})
		} else if d, ok := form["statements"]; ok {
			decls, _ = d.([]interface{})
		}
		for _, d := range decls {
			if decl, ok := d.(map[string]interface{}); ok {
				statements = append(statements, map[string]interface{}{
					"id":            decl["id"],
					"title":         decl["title"],
					"statement":     decl["statement"],
					"response_type": decl["response_type"],
					"options":       decl["options"],
				})
			}
		}
		formattedDeclarationForms = append(formattedDeclarationForms, map[string]interface{}{
			"id":         form["id"],
			"label":      form["name"],
			"statements": statements,
		})
	}

	resp := map[string]interface{}{
		"econsent_form_id":         form.ID,
		"research_id":              form.ResearchID,
		"title":                    form.Title,
		"version_number":           version.VersionNumber,     // Add version number here
		"econsent_form_version_id": version.ID,                // Add version ID here
		"video_description":        snapshot.VideoDescription, // from version
		"personal_info_check":      form.PersonalInfoCheck,    // from live
		"expiry_time":              form.ExpiryTime,           // from live
		"declaration_forms":        formattedDeclarationForms, // formatted as required
		"research_document":        researchDoc,               // from live, without mimeType
		"video":                    videoInfo,                 // merged
		"user": map[string]interface{}{
			"first_name": firstName,
			"last_name":  lastName,
			"gender":     gender,
			"nhi":        nhi,
			"dob":        dob,
		},
		//"isVerified":   isVerified,
		"assignmentId": assignmentID,
	}

	utils.JSONResponse(c, http.StatusOK, "E-consent form details fetched successfully", resp)
}

func GetConsentSessionResearchDocument(c *gin.Context) {

	session, ok := middlewares.GetConsentSession(c)

	if !ok {
		utils.JSONResponse(
			c,
			http.StatusUnauthorized,
			"Invalid consent session",
			nil,
		)
		return
	}

	// Get form from session
	var form models.EConsentForm

	err := config.DB.
		Where(
			"id = ? AND is_deleted = ?",
			session.EConsentFormID,
			false,
		).
		First(&form).Error

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusNotFound,
			"E-consent form not found",
			nil,
		)
		return
	}

	// Get research document
	var doc models.ResearchDocument

	err = config.DB.
		Joins("JOIN researches ON research_documents.research_id = researches.id").
		Where(
			"research_documents.research_id = ? AND research_documents.is_deleted = ? AND researches.is_deleted = ?",
			form.ResearchID,
			false,
			false,
		).
		First(&doc).Error

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			utils.JSONResponse(
				c,
				http.StatusNotFound,
				"Research document not found",
				nil,
			)
		} else {
			utils.JSONResponse(
				c,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to fetch research document: %v",
					err,
				),
				nil,
			)
		}

		return
	}

	// Check if file exists
	if _, err := os.Stat(doc.FilePath); os.IsNotExist(err) {
		utils.JSONResponse(
			c,
			http.StatusNotFound,
			"Document file not found on server",
			nil,
		)
		return
	}

	// Get file info
	fileInfo, err := os.Stat(doc.FilePath)

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusInternalServerError,
			"Failed to get file information",
			nil,
		)
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header(
		"Content-Disposition",
		fmt.Sprintf(
			"inline; filename=%s",
			filepath.Base(doc.FilePath),
		),
	)
	c.Header("Content-Type", doc.FileType)
	c.Header(
		"Content-Length",
		fmt.Sprintf("%d", fileInfo.Size()),
	)
	c.Header("Accept-Ranges", "bytes")

	c.File(doc.FilePath)
}

func SendConsentSessionVerificationCode(c *gin.Context) {

	session, ok := middlewares.GetConsentSession(c)

	if !ok {
		utils.JSONResponse(
			c,
			http.StatusUnauthorized,
			"Invalid consent session",
			nil,
		)
		return
	}

	assignmentID := session.AssignmentID

	var assignment models.EConsentAssignment

	err := config.DB.
		Preload("User").
		First(&assignment, assignmentID).Error

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusNotFound,
			"Assignment not found",
			nil,
		)
		return
	}

	if assignment.User.Email == "" {
		utils.JSONResponse(
			c,
			http.StatusBadRequest,
			"Patient email not found",
			nil,
		)
		return
	}

	cooldownKey := fmt.Sprintf(
		"consent_verification_cooldown:%d",
		assignment.ID,
	)

	limitKey := fmt.Sprintf(
		"consent_verification_limit:%d",
		assignment.ID,
	)

	exists, err := config.RedisClient.
		Exists(config.Ctx, cooldownKey).
		Result()

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusInternalServerError,
			"Rate limit check failed",
			nil,
		)
		return
	}

	if exists > 0 {
		utils.JSONResponse(
			c,
			http.StatusTooManyRequests,
			"Please wait 30 seconds before requesting another verification code",
			nil,
		)
		return
	}

	count, err := config.RedisClient.
		Incr(config.Ctx, limitKey).
		Result()

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusInternalServerError,
			"Rate limit check failed",
			nil,
		)
		return
	}

	if count == 1 {
		config.RedisClient.
			Expire(config.Ctx, limitKey, 15*time.Minute)
	}

	if count > 5 {
		utils.JSONResponse(
			c,
			http.StatusTooManyRequests,
			"Too many verification code requests. Please try again later.",
			nil,
		)
		return
	}

	config.RedisClient.Set(
		config.Ctx,
		cooldownKey,
		"1",
		30*time.Second,
	)

	otp := fmt.Sprintf(
		"%06d",
		rand.Intn(1000000),
	)

	otpKey := fmt.Sprintf(
		"consent_verification:%d",
		assignment.ID,
	)

	err = config.RedisClient.Set(
		config.Ctx,
		otpKey,
		otp,
		5*time.Minute,
	).Err()

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusInternalServerError,
			"Failed to store verification code",
			nil,
		)
		return
	}

	emailBody := emails.VerificationCodeEmail(
		assignment.User.FirstName,
		assignment.User.LastName,
		otp,
	)

	if err := config.SendEmail(
		[]string{assignment.User.Email},
		"E-Consent Verification Code",
		emailBody,
	); err != nil {

		config.RedisClient.Del(
			config.Ctx,
			otpKey,
		)

		utils.JSONResponse(
			c,
			http.StatusInternalServerError,
			"Failed to send verification email",
			nil,
		)
		return
	}

	utils.JSONResponse(
		c,
		http.StatusOK,
		"Verification code sent successfully",
		nil,
	)
}

func VerifyConsentSessionVerificationCode(c *gin.Context) {

	session, ok := middlewares.GetConsentSession(c)

	if !ok {
		utils.JSONResponse(
			c,
			http.StatusUnauthorized,
			"Invalid consent session",
			nil,
		)
		return
	}

	type VerifyRequest struct {
		Code string `json:"code"`
	}

	var req VerifyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONResponse(
			c,
			http.StatusBadRequest,
			"Invalid request body",
			nil,
		)
		return
	}

	var assignment models.EConsentAssignment

	err := config.DB.
		First(
			&assignment,
			session.AssignmentID,
		).Error

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusNotFound,
			"Assignment not found",
			nil,
		)
		return
	}

	key := fmt.Sprintf(
		"consent_verification:%d",
		assignment.ID,
	)

	storedOTP, err := config.RedisClient.
		Get(config.Ctx, key).
		Result()

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusBadRequest,
			"Verification code expired or invalid",
			nil,
		)
		return
	}

	if storedOTP != req.Code {
		utils.JSONResponse(
			c,
			http.StatusBadRequest,
			"Invalid verification code",
			nil,
		)
		return
	}

	assignment.IsVerified = true

	err = config.DB.Save(
		&assignment,
	).Error

	if err != nil {
		utils.JSONResponse(
			c,
			http.StatusInternalServerError,
			"Failed to update verification status",
			nil,
		)
		return
	}

	config.RedisClient.Del(
		config.Ctx,
		key,
	)

	utils.JSONResponse(
		c,
		http.StatusOK,
		"Verification successful",
		nil,
	)
}

func CreateEConsentUsingSession(c *gin.Context) {
	session, ok := middlewares.GetConsentSession(c)

	if !ok {
		utils.JSONResponse(
			c,
			http.StatusUnauthorized,
			"Invalid consent session",
			nil,
		)
		return
	}
	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start transaction", nil)
		return
	}

	// Parse form fields
	eConsentFormID := session.EConsentFormID
	nhi := c.PostForm("nhi")
	firstName := c.PostForm("first_name")
	lastName := c.PostForm("last_name")
	gender := c.PostForm("gender")
	dobStr := c.PostForm("dob")
	declarationResponsesStr := c.PostForm("declaration_responses")

	patientID := session.PatientID
	nurseId := session.NurseID
	// Validate required fields
	missingFields := []string{}
	if nhi == "" {
		missingFields = append(missingFields, "NHI")
	}
	if firstName == "" {
		missingFields = append(missingFields, "First Name")
	}
	if lastName == "" {
		missingFields = append(missingFields, "Last Name")
	}
	if gender == "" {
		missingFields = append(missingFields, "Gender")
	}
	if dobStr == "" {
		missingFields = append(missingFields, "Dob")
	}
	if declarationResponsesStr == "" {
		missingFields = append(missingFields, "declarations")
	}
	if len(missingFields) > 0 {
		tx.Rollback()
		msg := "Missing required fields: " + strings.Join(missingFields, ", ")
		utils.JSONResponse(c, http.StatusBadRequest, msg, nil)
		return
	}

	// Decrypt and validate fields
	decryptedNHI, err := utils.Decrypt(nhi)
	if err != nil || decryptedNHI == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted NHI", nil)
		return
	}
	decryptedFirstName, err := utils.Decrypt(firstName)
	if err != nil || decryptedFirstName == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted first name", nil)
		return
	}
	decryptedLastName, err := utils.Decrypt(lastName)
	if err != nil || decryptedLastName == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted last name", nil)
		return
	}
	decryptedGender, err := utils.Decrypt(gender)
	if err != nil || decryptedGender == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted gender", nil)
		return
	}
	decryptedDOB, err := utils.Decrypt(dobStr)
	if err != nil || decryptedDOB == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted DOB", nil)
		return
	}

	// Parse new declaration_responses format (array of FormDeclarationInput)
	var declarationResponsesRaw []map[string]interface{}
	if err := json.Unmarshal([]byte(declarationResponsesStr), &declarationResponsesRaw); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid declaration_responses JSON", nil)
		return
	}
	if len(declarationResponsesRaw) == 0 {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "At least one declaration response is required", nil)
		return
	}

	var formDeclarationInputs []FormDeclarationInput
	for _, formItem := range declarationResponsesRaw {
		formIDRaw, okFormID := formItem["form_id"].(float64)
		declArr, okDeclArr := formItem["declaration"].([]interface{})
		if !okFormID || !okDeclArr {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, "Invalid form declaration format", nil)
			return
		}
		formID := uint(formIDRaw)
		var declarations []DeclarationResponseInput
		for _, declItem := range declArr {
			declMap, ok := declItem.(map[string]interface{})
			if !ok {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid declaration item format", nil)
				return
			}
			declIDRaw, okID := declMap["declaration_id"].(float64)
			respRaw, okResp := declMap["response"]
			if !okID || !okResp {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid declaration response format", nil)
				return
			}
			declIDUint := uint(declIDRaw)
			// If response is string, decrypt; if bool, keep as is
			var respVal interface{}
			switch v := respRaw.(type) {
			case string:
				respDecrypted, err := utils.Decrypt(v)
				if err != nil || respDecrypted == "" {
					tx.Rollback()
					utils.JSONResponse(c, http.StatusBadRequest, "Invalid or missing encrypted response", nil)
					return
				}
				// Try to parse as bool, else keep as string
				if respDecrypted == "true" || respDecrypted == "1" {
					respVal = true
				} else if respDecrypted == "false" || respDecrypted == "0" {
					respVal = false
				} else {
					respVal = respDecrypted
				}
			case bool:
				respVal = v
			default:
				tx.Rollback()
				utils.JSONResponse(c, http.StatusBadRequest, "Invalid response type", nil)
				return
			}
			declarations = append(declarations, DeclarationResponseInput{
				DeclarationID: declIDUint,
				Response:      respVal,
			})
		}
		formDeclarationInputs = append(formDeclarationInputs, FormDeclarationInput{
			FormID:      formID,
			Declaration: declarations,
		})
	}

	// Convert []FormDeclarationInput to []services.FormDeclarationInput
	var serviceFormDeclarations []services.FormDeclarationInput
	for _, formInput := range formDeclarationInputs {
		var serviceDecls []services.DeclarationResponseInput
		for _, dr := range formInput.Declaration {
			serviceDecls = append(serviceDecls, services.DeclarationResponseInput{
				DeclarationID: dr.DeclarationID,
				Response:      dr.Response, // pass as-is (string or bool)
			})
		}
		serviceFormDeclarations = append(serviceFormDeclarations, services.FormDeclarationInput{
			FormID:      formInput.FormID,
			Declaration: serviceDecls,
		})
	}

	// Get the latest version for the form
	var latestVersion models.EConsentFormVersion
	err = config.DB.Where("e_consent_form_id = ?", uint(eConsentFormID)).Order("version_number DESC").First(&latestVersion).Error
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusNotFound, "E-consent version not found", nil)
		return
	}

	// Find the assignment for this patient, form, and latest version
	var assignment models.EConsentAssignment
	err = config.DB.Where("e_consent_form_id = ? AND patient_id = ? AND e_consent_form_version_id = ? AND is_deleted = ?", session.EConsentFormID, session.PatientID, latestVersion.ID, false).First(&assignment).Error
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "No e-consent assignment found for this patient and version", nil)
		return
	}
	// return
	// Use the latest version ID for the rest of the logic
	eConsentFormVersionID := latestVersion.ID

	var patient models.User

	err = config.DB.
		Where("id = ?", patientID).
		First(&patient).Error

	if err != nil {
		tx.Rollback()
		utils.JSONResponse(
			c,
			http.StatusBadRequest,
			"Patient not found",
			nil,
		)
		return
	}

	// Call the service with the new format
	err = services.CreatePatientEConsentService(
		tx, &patient, patientID, nurseId, uint(eConsentFormID), eConsentFormVersionID, decryptedNHI, decryptedFirstName, decryptedLastName, decryptedGender, decryptedDOB, serviceFormDeclarations, c,
	)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// Notify all coordinators (nurses) that patient has signed
	var eConsentForm models.EConsentForm
	if err := config.DB.Preload("Research").Where("id = ?", uint(eConsentFormID)).First(&eConsentForm).Error; err == nil {
		for _, coordinatorID := range eConsentForm.CoordinatorIDs {
			// Notification
			_ = services.CreateNotification(
				config.DB,
				coordinatorID,
				"Patient Signed E-Consent",
				fmt.Sprintf("Patient %s %s has signed their e-consent form.", decryptedFirstName, decryptedLastName),
				"E-Consent",
				decryptedFirstName+" "+decryptedLastName,
				"econsents",
				uint(eConsentFormID),
				map[string]interface{}{
					"econsent_id": uint(eConsentFormID),
					"status":      "signed",
				},
			)

			// Email
			var coordinator models.User
			if err := config.DB.Where("id = ?", coordinatorID).First(&coordinator).Error; err == nil && coordinator.Email != "" {
				emailBody := fmt.Sprintf(
					"Dear %s,\n\nPatient %s %s has signed the e-consent form \"%s\" for research \"%s\".\n\nBest regards,\nTheranostics Team",
					coordinator.FirstName,
					decryptedFirstName,
					decryptedLastName,
					eConsentForm.Title,
					eConsentForm.Research.Title,
				)
				subject := "Patient Signed E-Consent Notification"
				_ = config.SendEmail([]string{coordinator.Email}, subject, emailBody)
			}
		}
	}

	utils.JSONResponse(c, http.StatusOK, "Patient e-consent created successfully", nil)
}
