package services

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"
	"time"

	"gorm.io/gorm"
)

// KitRegistrationResponse represents the response for kit registration list
type KitRegistrationResponse struct {
	ID               uint      `json:"id"`
	PatientFirstName string    `json:"patient_first_name"`
	PatientLastName  string    `json:"patient_last_name"`
	PatientEmail     string    `json:"patient_email"`
	PatientAge       string    `json:"patient_age"`
	PatientGender    string    `json:"patient_gender"`
	KitStatus        string    `json:"kit_status"`
	CreatedAt        time.Time `json:"created_at"`
	BarcodeNumber    string    `json:"barcode_number"`
	FilePath         string    `json:"file_path"`
	Reason           string    `json:"reason"`
	// Added customer details
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	// Added order details
	OrderNumber string `json:"order_number"`
}

// PatientInfo represents the decrypted patient details
type PatientInfo struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Age       string `json:"age"`
	Gender    string `json:"gender"`
}

// validatePaginationParams validates and returns page and per_page values
func ValidatePaginationParams(query url.Values) (int, int, error) {
	// Default and validate 'page'
	page := 1
	if val := query.Get("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		} else {
			return 0, 0, errors.New(utils.MsgInvalidPageParameter)
		}
	}

	// Default and validate 'per_page'
	perPage := 10
	if val := query.Get("per_page"); val != "" {
		if pp, err := strconv.Atoi(val); err == nil && pp > 0 {
			perPage = pp
		} else {
			return 0, 0, errors.New(utils.MsgInvalidPerPageParameter)
		}
	}

	return page, perPage, nil
}

// validateSortParams validates and returns sort column and sort order
func ValidateSortParams(query url.Values) (string, string, error) {
	// Default and validate 'sort'
	sort := "desc"
	if val := strings.ToLower(query.Get("sort")); val == "asc" || val == "desc" {
		sort = val
	} else if val != "" {
		return "", "", errors.New(utils.MsgInvalidSortParameter)
	}

	// Default and validate 'sort_column'
	sortColumn := "created_at"
	validSortColumns := []string{"created_at", "kit_status", "barcode_number"}
	if val := strings.ToLower(query.Get("sort_column")); val != "" {
		if utils.StringInSlice(val, validSortColumns) {
			sortColumn = val
		} else {
			return "", "", errors.New(utils.MsgInvalidSortColumnParameter)
		}
	}

	return sortColumn, sort, nil
}

// validateStatusParam validates the status parameter and returns the status or an error.
func ValidateStatusParam(query url.Values) (string, error) {
	status := "all" // Default value for status
	val := query.Get("status")
	if val == "Not-Received" || val == "Received" || val == "Reject" || val == "Send" || val == "all" {
		status = val
		return status, nil
	}

	if val == "" {
		return status, nil
	}
	return "", errors.New(utils.MsgInvalidKitStatusParmas)
}

// fetchKitRegistrations retrieves and processes kit registrations
func FetchKitRegistrations(page, perPage int, sortColumn, sort, searchText, status string) (int, []KitRegistrationResponse, error) {
	var registrations []models.KitRegistration
	var totalCount int64

	// Calculate offset
	offset := (page - 1) * perPage

	// Base query
	query := config.DB.Model(&models.KitRegistration{}).
		Preload("Barcode").
		Preload("Customer").
		Preload("Order").
		Where("is_deleted = ?", false)

	// Apply status filter if not "all"
	if status != "all" {
		query = query.Where("kit_status = ?", status)
	}

	// Encrypt search text for encrypted patient fields
	if searchText != "" {
		encryptedSearchText, err := utils.Encrypt((searchText))
		if err != nil {
			return 0, nil, fmt.Errorf("failed to encrypt search text: %v", err)
		}
		encryptedSearchPattern := "%" + encryptedSearchText + "%"
		decryptedSearchPattern := "%" + searchText + "%"
		query = query.Where(`
		EXISTS (SELECT 1 FROM customers c 
			WHERE kit_registrations.kit_customer_id = c.id 
			AND (LOWER(c.first_name) LIKE LOWER(?) 
				OR LOWER(c.last_name) LIKE LOWER(?) 
				OR LOWER(c.email) LIKE LOWER(?)))
		OR EXISTS (SELECT 1 FROM orders o 
			WHERE kit_registrations.kit_order_id = o.id 
			AND LOWER(o.order_number) LIKE LOWER(?))
		OR EXISTS (SELECT 1 FROM barcodes b 
			WHERE kit_registrations.kit_barcode_order_id = b.id 
			AND LOWER(b.barcode_number) LIKE LOWER(?))
		OR LOWER(kit_registrations.patient_first_name) LIKE LOWER(?)
		OR LOWER(kit_registrations.patient_last_name) LIKE LOWER(?)
		OR LOWER(kit_registrations.patient_email) LIKE LOWER(?)
		OR LOWER(kit_registrations.patient_gender) LIKE LOWER(?)
		OR LOWER(kit_registrations.patient_age) LIKE LOWER(?)
		OR CAST(kit_registrations.created_at AS TEXT) ILIKE ?`,
			decryptedSearchPattern, decryptedSearchPattern, decryptedSearchPattern,
			decryptedSearchPattern, decryptedSearchPattern,
			encryptedSearchPattern, encryptedSearchPattern, encryptedSearchPattern, encryptedSearchPattern, encryptedSearchPattern, decryptedSearchPattern)
	}
	// Get total count before pagination
	if err := query.Count(&totalCount).Error; err != nil {
		return 0, nil, err
	}

	// Apply sorting
	switch sortColumn {
	case "customer_name":
		query = query.Joins("JOIN customers ON kit_registrations.kit_customer_id = customers.id").
			Order(fmt.Sprintf("customers.first_name %s, customers.last_name %s", sort, sort))
	case "customer_email":
		query = query.Joins("JOIN customers ON kit_registrations.kit_customer_id = customers.id").
			Order(fmt.Sprintf("customers.email %s", sort))
	case "order_number":
		query = query.Joins("JOIN orders ON kit_registrations.kit_order_id = orders.id").
			Order(fmt.Sprintf("orders.order_number %s", sort))
	case "barcode_number":
		query = query.Joins("JOIN barcodes ON kit_registrations.kit_barcode_order_id = barcodes.id").
			Order(fmt.Sprintf("barcodes.barcode_number %s", sort))
	case "created_at":
		query = query.Order(fmt.Sprintf("created_at %s", sort))
	case "kit_status":
		query = query.Order(fmt.Sprintf("kit_status %s", sort))
	default:
		query = query.Order(fmt.Sprintf("created_at %s", sort))
	}

	// Apply pagination
	if err := query.Limit(perPage).Offset(offset).Find(&registrations).Error; err != nil {
		return 0, nil, err
	}

	// Map to response struct
	var response []KitRegistrationResponse
	for _, reg := range registrations {
		// Construct customer name
		customerName := utils.CapitalizeWords(fmt.Sprintf("%s %s", reg.Customer.FirstName, reg.Customer.LastName))

		// Create response object
		kitResponse := KitRegistrationResponse{
			ID:               reg.ID,
			PatientFirstName: reg.PatientFirstName,
			PatientLastName:  reg.PatientLastName,
			PatientEmail:     reg.PatientEmail,
			PatientAge:       reg.PatientAge,
			PatientGender:    reg.PatientGender,
			KitStatus:        reg.KitStatus,
			CreatedAt:        reg.CreatedAt,
			BarcodeNumber:    reg.Barcode.BarcodeNumber,
			FilePath:         reg.FilePath,
			Reason:           reg.Reason,
			CustomerName:     customerName,
			CustomerEmail:    reg.Customer.Email,
			OrderNumber:      reg.Order.OrderNumber,
		}

		response = append(response, kitResponse)
	}

	return int(totalCount), response, nil
}

// CheckPatientExists checks if the patient exists in the database
func CheckPatientExists(patientID uint) (bool, error) {
	var patient models.KitRegistration // Assuming you have a Patient model

	// Check if the patient exists in the database
	err := config.DB.Model(&models.KitRegistration{}).Where("id = ?", patientID).First(&patient).Error

	// If no record is found, return false
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	// If record is found, return true
	return true, nil
}

// validateStatusTransition checks if the status transition is valid
func ValidateStatusTransition(current, new, reason string) error {
	current = strings.ToLower(current)
	new = strings.ToLower(new)

	switch current {
	case "not-received":
		if new == "received" || new == "reject" || new == "send" {
			if new == "reject" && strings.TrimSpace(reason) == "" {
				return errors.New("Reason is mandatory for reject status")
			}
			return nil
		}
		return errors.New("Invalid transition: 'not received' can only change to 'received', 'reject' or 'send'")

	case "received":
		if new == "not-received" {
			return errors.New("Invalid transition: 'received' cannot change to 'not received'")
		}
		return nil

	case "reject":
		return errors.New("Invalid transition: 'reject' cannot change to any other status")

	case "send":
		return errors.New("Invalid transition: 'send' cannot change to any other status")

	default:
		return errors.New("Unknown current status")
	}
}

func ValidateSendStatus(Type string, LabName string, LabAddress string, NHINumber string, LabID uint) error {
	if Type != "new" && Type != "old" {
		return errors.New("Type must be either 'new' or 'old' for send status")
	}

	if Type == "old" {
		if LabID == 0 {
			return errors.New("Lab ID is required when type is 'old'")
		}
	} else {
		// Validate Lab Name
		LabName = strings.TrimSpace(LabName)
		if len(LabName) < 5 {
			return errors.New("Lab name must have at least 5 characters")
		}
		if len(LabName) > 255 {
			return errors.New("Lab name cannot exceed 255 characters")
		}

		// Validate Lab Address
		LabAddress = strings.TrimSpace(LabAddress)
		if len(LabAddress) < 5 {
			return errors.New("Lab address must have at least 5 characters")
		}
		if len(LabAddress) > 255 {
			return errors.New("Lab address cannot exceed 255 characters")
		}

		// Validate NHI Number
		NHINumber = strings.TrimSpace(NHINumber)
		if NHINumber == "" {
			return errors.New("NHI number is required when type is 'new'")
		}
		if len(NHINumber) > 10 {
			return errors.New("NHI number cannot exceed 10 characters")
		}

		// Ensure NHI number is alphanumeric with optional '-'
		alphanumericWithDash := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
		if !alphanumericWithDash.MatchString(NHINumber) {
			return errors.New("NHI number must be alphanumeric and may include '-'")
		}
	}
	return nil
}

func ProcessLabInformation(Type string, LabName string, LabAddress string, NHINumber string, LabID uint) (uint, error) {
	if Type == "old" {
		// Verify if lab exists
		exists, err := checkLabExists(LabID)
		if err != nil {
			return 0, err
		}
		if !exists {
			return 0, errors.New("Specified lab ID does not exist")
		}
		return LabID, nil
	}

	// For new lab, check if lab with same details exists
	existingLab, err := findExistingLab(LabName, LabAddress, NHINumber)
	if err != nil {
		return 0, err
	}

	if existingLab != nil {
		return existingLab.ID, nil
	}

	// Create new lab
	lab := &models.Lab{
		LabName:    LabName,
		LabAddress: LabAddress,
		NHINumber:  NHINumber,
	}

	if err := config.DB.Create(lab).Error; err != nil {
		return 0, fmt.Errorf("failed to create new lab: %w", err)
	}

	return lab.ID, nil
}

func checkLabExists(labID uint) (bool, error) {
	var count int64
	err := config.DB.Model(&models.Lab{}).Where("id = ? AND is_deleted = ?", labID, false).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check lab existence: %w", err)
	}
	return count > 0, nil
}

func findExistingLab(name, address, nhiNumber string) (*models.Lab, error) {
	var lab models.Lab
	err := config.DB.Where("lab_name = ? AND lab_address = ? AND nhi_number = ? AND is_deleted = ?",
		name, address, nhiNumber, false).First(&lab).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check existing lab: %w", err)
	}
	return &lab, nil
}

// GetPatientStatus fetches the current status of the patient from the database
func GetPatientStatus(patientID uint) (string, error) {

	var patient models.KitRegistration

	// Get the patient’s record from the database
	err := config.DB.Model(&models.KitRegistration{}).Where("id = ?", patientID).First(&patient).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", errors.New("Patient not found")
		}
		return "", err
	}

	return patient.KitStatus, nil // Assuming the Patient model has a 'Status' field
}

// UpdatePatientStatus updates the status of the patient in the database
func UpdatePatientStatus(patientID uint, newStatus, reason string, labID uint) error {
	// Start a transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("Panic recovered: %v", r)
		}
	}()

	// Find the patient by ID
	var patient models.KitRegistration
	if err := tx.Model(&models.KitRegistration{}).Where("id = ?", patientID).First(&patient).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Decrypt individual patient details
	decryptedFirstName, err := utils.Decrypt(patient.PatientFirstName)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to decrypt patient first name for ID %d: %w", patient.ID, err)
	}

	decryptedLastName, err := utils.Decrypt(patient.PatientLastName)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to decrypt patient last name for ID %d: %w", patient.ID, err)
	}

	decryptedGender, err := utils.Decrypt(patient.PatientGender)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to decrypt patient gender for ID %d: %w", patient.ID, err)
	}

	decryptedAge, err := utils.Decrypt(patient.PatientAge)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to decrypt patient age for ID %d: %w", patient.ID, err)
	}

	decryptedEmail, err := utils.Decrypt(patient.PatientEmail)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to decrypt patient email for ID %d: %w", patient.ID, err)
	}

	patientInfo := PatientInfo{
		FirstName: decryptedFirstName,
		LastName:  decryptedLastName,
		Age:       decryptedAge,
		Gender:    decryptedGender,
		Email:     decryptedEmail,
	}
	// Normalize status
	switch strings.ToLower(newStatus) {
	case "not-received":
		newStatus = "Not-Received"
	case "received":
		newStatus = "Received"
	case "send":
		newStatus = "Send"
	default:
		newStatus = "Reject"
	}

	// Update the status and reason
	patient.KitStatus = newStatus
	patient.Reason = reason

	if labID > 0 {
		patient.LabID = labID
	}

	// Fetch full customer details for email
	var customer models.Customer
	if err := tx.Model(&customer).Where("id = ?", patient.KitCustomerID).First(&customer).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Fetch full order details for email
	var order models.Order
	if err := tx.Model(&order).Where("id = ?", patient.KitOrderID).First(&order).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Fetch full order details for email
	var barcode models.Barcode
	if err := tx.Model(&barcode).Where("id = ?", patient.KitBarcodeOrderID).First(&barcode).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Save the updated patient record
	if err := tx.Save(&patient).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction before sending emails
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send email notifications (outside of transaction to avoid delaying response)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in email sending goroutine: %v", r)
			}
		}()

		// Admin Notification
		adminEmailContent := emails.AdminStatusUpdateNotificationEmail(
			customer.FirstName,
			customer.LastName,
			customer.Email,
			customer.PhoneNumber,
			customer.StreetAddress,
			customer.TownCity,
			customer.Region,
			customer.Postcode,
			customer.Country,
			customer.ShippingAddress,
			customer.ShippingTownCity,
			customer.ShippingRegion,
			customer.ShippingPostcode,
			customer.ShippingCountry,
			order.OrderNumber,
			customer.ClinicID,
			order.ProductName,
			order.TrackingID,
			order.ProductDescription,
			order.Quantity,
			order.ProductPrice,
			order.ProductGstPrice,
			order.ProductDiscount,
			order.TotalPrice,
			barcode.BarcodeNumber,
			patientInfo.FirstName,
			patientInfo.LastName,
			patientInfo.Email,
			patientInfo.Gender,
			patientInfo.Age,
			patient.KitStatus, patient.Reason,
		)
		err := config.SendEmail(
			[]string{config.AppConfig.ClientEmail},
			"Kit Status Update",
			adminEmailContent,
		)
		if err != nil {
			log.Printf("Failed to send admin notification email: %v", err)
		}

		if patient.IsClinicInform { // Customer Notification
			customerEmailContent := emails.CustomerStatusUpdateNotificationEmail(
				customer.FirstName,
				customer.LastName,
				customer.Email,
				customer.PhoneNumber,
				customer.StreetAddress,
				customer.TownCity,
				customer.Region,
				customer.Postcode,
				customer.Country,
				customer.ShippingAddress,
				customer.ShippingTownCity,
				customer.ShippingRegion,
				customer.ShippingPostcode,
				customer.ShippingCountry,
				order.OrderNumber,
				customer.ClinicID,
				order.ProductName,
				order.TrackingID,
				order.ProductDescription,
				order.Quantity,
				order.ProductPrice,
				order.ProductGstPrice,
				order.ProductDiscount,
				order.TotalPrice,
				barcode.BarcodeNumber,
				patientInfo.FirstName,
				patientInfo.LastName,
				patientInfo.Email,
				patientInfo.Gender,
				patientInfo.Age,
				patient.KitStatus, patient.Reason,
			)
			err = config.SendEmail(
				[]string{customer.Email},
				"Kit Status Update",
				customerEmailContent,
			)
		}
		if err != nil {
			log.Printf("Failed to send customer notification email: %v", err)
		}

		// Customer Notification
		patientEmailContent := emails.CustomerStatusUpdateNotificationEmail(
			customer.FirstName,
			customer.LastName,
			customer.Email,
			customer.PhoneNumber,
			customer.StreetAddress,
			customer.TownCity,
			customer.Region,
			customer.Postcode,
			customer.Country,
			customer.ShippingAddress,
			customer.ShippingTownCity,
			customer.ShippingRegion,
			customer.ShippingPostcode,
			customer.ShippingCountry,
			order.OrderNumber,
			customer.ClinicID,
			order.ProductName,
			order.TrackingID,
			order.ProductDescription,
			order.Quantity,
			order.ProductPrice,
			order.ProductGstPrice,
			order.ProductDiscount,
			order.TotalPrice,
			barcode.BarcodeNumber,
			patientInfo.FirstName,
			patientInfo.LastName,
			patientInfo.Email,
			patientInfo.Gender,
			patientInfo.Age,
			patient.KitStatus, patient.Reason,
		)
		err = config.SendEmail(
			[]string{patientInfo.Email},
			"Kit Status Update",
			patientEmailContent,
		)
		if err != nil {
			log.Printf("Failed to send customer notification email: %v", err)
		}

	}()

	return nil
}

// Separate function to retrieve customer data outside of the transaction
func retrieveCustomerData(customerID uint) (*models.Customer, error) {
	var customer models.Customer
	result := config.DB.First(&customer, "id = ? AND is_deleted = ?", customerID, false)
	if result.Error != nil {
		return nil, fmt.Errorf(utils.MsgCustomerNotFound)
	}
	return &customer, nil
}

func HandleFileUpload(patientID string, file io.Reader, filename, fileExt string) error {
	// Start a new transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %v", tx.Error)
	}

	// Use a defer to handle transaction rollback in case of panic
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Validate patient registration exists and kit status is received
	var kitRegistration models.KitRegistration
	result := tx.Where("id = ? AND kit_status = ?", patientID, "Send").First(&kitRegistration)
	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("Patient registration not found or kit not received: %v", result.Error)
	}

	// Generate unique filename
	currentDate := time.Now().Format("20060102")
	uniqueID := utils.GenerateUniqueID()
	newFilename := fmt.Sprintf("%s_%s%s", currentDate, uniqueID, fileExt)

	// Define upload directory
	uploadDir := "public/files"
	filePath := filepath.Join(uploadDir, newFilename)

	// Delete existing file if exists
	if kitRegistration.FilePath != "" {
		os.Remove(filepath.Join(uploadDir, kitRegistration.FilePath))
	}

	// Create new file
	dst, err := os.Create(filePath)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	// Copy uploaded file contents
	_, err = io.Copy(dst, file)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save file: %v", err)
	}

	// Update kit registration with new file path
	updateResult := tx.Model(&kitRegistration).Update("file_path", newFilename)
	if updateResult.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update file path: %v", updateResult.Error)
	}

	// Commit transaction first
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Retrieve customer data outside of the transaction
	customer, err := retrieveCustomerData(kitRegistration.KitCustomerID)
	if err != nil {
		log.Printf("Failed to retrieve customer data: %v", err)
		return nil
	}

	// Decrypt individual patient details
	decryptedFirstName, err := utils.Decrypt(kitRegistration.PatientFirstName)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to decrypt patient first name for ID %d: %w", kitRegistration.ID, err)
	}

	decryptedLastName, err := utils.Decrypt(kitRegistration.PatientLastName)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to decrypt patient last name for ID %d: %w", kitRegistration.ID, err)
	}

	decryptedEmail, err := utils.Decrypt(kitRegistration.PatientEmail)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to decrypt patient last name for ID %d: %w", kitRegistration.ID, err)
	}

	patientData := patientDataReponse{
		FirstName: decryptedFirstName,
		LastName:  decryptedLastName,
		Email:     decryptedEmail,
	}

	// Construct report link
	reportLink := fmt.Sprintf("%s/files/%s", config.AppConfig.ApiUrl, newFilename)

	if kitRegistration.IsClinicInform { // Prepare and send email
		emailContent := emails.ReportUploadNotificationEmail(
			customer.FirstName,
			customer.LastName,
			patientData.FirstName,
			patientData.LastName,
			reportLink,
		)

		err = config.SendEmail(
			[]string{customer.Email},
			"Theranostics Lab Report",
			emailContent,
		)

		if err != nil {
			log.Printf("Failed to send report upload email: %v", err)
		}
	}

	// Prepare and send email
	patientEmailContent := emails.ReportUploadNotificationEmail(
		patientData.FirstName,
		patientData.LastName,
		patientData.FirstName,
		patientData.LastName,
		reportLink,
	)

	err = config.SendEmail(
		[]string{patientData.Email},
		"Theranostics Lab Report",
		patientEmailContent,
	)

	if err != nil {
		log.Printf("Failed to send report upload email: %v", err)
	}

	return nil
}
