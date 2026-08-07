package services

import (
	"fmt"

	"strings"
	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"

	"gorm.io/gorm"
)

type patientDataReponse struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type KitRegistration struct {
	ID                uint `gorm:"primaryKey"`
	KitCustomerID     uint
	KitOrderID        uint
	KitBarcodeOrderID uint
	FirstName         string
	LastName          string
	Gender            string
	Age               string
}

func ValidatePatientRequest(FirstName string, LastName string, Gender string, Age int, Email string) error {
	if !utils.IsValidFirstName(FirstName) {
		return fmt.Errorf(utils.MsgInvalidFirstName)
	}
	if LastName != "" && !utils.IsValidLastName(LastName) {
		return fmt.Errorf(utils.MsgInvalidLastName)
	}
	if !utils.IsValidEmail(Email) {
		return fmt.Errorf(utils.MsgInvalidEmail)
	}

	validGenders := []string{"male", "female", "other"}
	if !utils.Contains(validGenders, strings.ToLower(Gender)) {
		return fmt.Errorf(utils.MsgInvalidGender)
	}

	if Age < 18 {
		return fmt.Errorf(utils.MsgInvalidAge)
	}
	return nil
}

func ValidateBarcodeAndOrder(tx *gorm.DB, barcodeNumber string, orderID uint) (uint, error) {
	var barcode models.Barcode
	if err := tx.Where("barcode_number = ? AND order_id = ? AND is_deleted = ?", barcodeNumber, orderID, false).
		First(&barcode).Error; err != nil {
		return 0, fmt.Errorf(utils.MsgBarcodeOrOrderNotFound)
	}
	return barcode.ID, nil
}

func ValidateCustomerID(tx *gorm.DB, customerID uint) (*models.Customer, error) {
	var customer models.Customer
	if err := tx.First(&customer, "id = ? AND is_deleted = ?", customerID, false).Error; err != nil {
		return nil, fmt.Errorf(utils.MsgCustomerNotFound)
	}
	return &customer, nil
}

func ValidateOrderID(tx *gorm.DB, orderID uint) (*models.Order, error) {
	var order models.Order
	if err := tx.First(&order, "id = ? AND is_deleted = ?", orderID, false).Error; err != nil {
		return nil, fmt.Errorf(utils.MsgOrderNotFound)
	}
	return &order, nil
}

func IsKitRegistrationExists(tx *gorm.DB, barcodeOrderID, orderID, customerID uint) (bool, error) {
	var count int64
	err := tx.Model(&models.KitRegistration{}).
		Where("kit_barcode_order_id = ? AND kit_order_id = ? AND kit_customer_id = ? AND is_deleted = ?", barcodeOrderID, orderID, customerID, false).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func InsertKitRegistration(tx *gorm.DB, FirstName string, LastName string, Email string, Gender string, Age int, barcodeID uint, CustomerID uint, OrderID uint, IsClinicInform bool) error {

	// Encrypt individual patient details
	encryptedFirstName, err := utils.Encrypt(FirstName)
	if err != nil {
		return err
	}

	encryptedLastName, err := utils.Encrypt(LastName)
	if err != nil {
		return err
	}

	encryptedGender, err := utils.Encrypt(Gender)
	if err != nil {
		return err
	}

	encryptedAge, err := utils.Encrypt(fmt.Sprintf("%d", Age))
	if err != nil {
		return err
	}

	encryptedEmail, err := utils.Encrypt(Email)
	if err != nil {
		return err
	}

	// Create the kit registration record
	registration := models.KitRegistration{
		KitCustomerID:     CustomerID,
		KitOrderID:        OrderID,
		KitBarcodeOrderID: barcodeID,
		PatientFirstName:  encryptedFirstName,
		PatientLastName:   encryptedLastName,
		PatientGender:     encryptedGender,
		PatientAge:        encryptedAge,
		PatientEmail:      encryptedEmail,
		IsClinicInform:    IsClinicInform,
	}

	// Save to the database
	if err := tx.Create(&registration).Error; err != nil {
		return err
	}

	return nil
}

func SendKitRegistrationNotificationEmails(
	customerFirstName string,
	customerLastName string,
	customerEmail string,
	customerPhone string,
	customerStreetAddress string,
	customerTownCity string,
	customerRegion string,
	customerPostcode string,
	customerCountry string,
	customerShippingStreetAddress string,
	customerShippingTownCity string,
	customerShippingRegion string,
	customerShippingPostcode string,
	customerShippingCountry string,
	orderNumber string,
	clinicID string,
	productName string,
	productDescription string,
	productQuantity int,
	productPrice float64,
	productGSTPrice float64,
	productDiscount float64,
	totalPrice float64,
	trackingID string,
	barcodeNumber string,
	patientFirstName string,
	patientLastName string,
	patientGender string,
	patientEmail string,
	patientAge int, IsClinicInform bool) error {

	adminEmailContent := emails.AdminKitRegistrationNotificationEmail(customerFirstName,
		customerLastName,
		customerEmail,
		customerPhone,
		customerStreetAddress,
		customerTownCity,
		customerRegion,
		customerPostcode,
		customerCountry,
		customerShippingStreetAddress,
		customerShippingTownCity,
		customerShippingRegion,
		customerShippingPostcode,
		customerShippingCountry,
		orderNumber,
		trackingID,
		clinicID,
		productName,
		productDescription,
		productQuantity,
		productPrice,
		productGSTPrice,
		productDiscount,
		totalPrice,
		barcodeNumber,
		patientFirstName,
		patientLastName,
		patientGender,
		patientEmail,
		patientAge)
	err := config.SendEmail([]string{config.AppConfig.ClientEmail}, "New Kit Registration", adminEmailContent)
	if err != nil {
		return fmt.Errorf("failed to send admin notification email: %v", err)
	}

	if IsClinicInform {
		customerEmailContent := emails.CustomerKitRegistrationNotificationEmail(customerFirstName,
			customerLastName,
			customerEmail,
			customerPhone,
			customerStreetAddress,
			customerTownCity,
			customerRegion,
			customerPostcode,
			customerCountry,
			customerShippingStreetAddress,
			customerShippingTownCity,
			customerShippingRegion,
			customerShippingPostcode,
			customerShippingCountry,
			orderNumber,
			trackingID,
			clinicID,
			productName,
			productDescription,
			productQuantity,
			productPrice,
			productGSTPrice,
			productDiscount,
			totalPrice,
			barcodeNumber,
			patientFirstName,
			patientLastName,
			patientGender,
			patientEmail,
			patientAge)
		err = config.SendEmail([]string{customerEmail}, "Kit Registration Confirmation", customerEmailContent)
		if err != nil {
			return fmt.Errorf("failed to send customer notification email: %v", err)
		}
	}

	patientEmailContent := emails.CustomerKitRegistrationNotificationEmail(customerFirstName,
		customerLastName,
		customerEmail,
		customerPhone,
		customerStreetAddress,
		customerTownCity,
		customerRegion,
		customerPostcode,
		customerCountry,
		customerShippingStreetAddress,
		customerShippingTownCity,
		customerShippingRegion,
		customerShippingPostcode,
		customerShippingCountry,
		orderNumber,
		trackingID,
		clinicID,
		productName,
		productDescription,
		productQuantity,
		productPrice,
		productGSTPrice,
		productDiscount,
		totalPrice,
		barcodeNumber,
		patientFirstName,
		patientLastName,
		patientGender,
		patientEmail,
		patientAge)
	err = config.SendEmail([]string{patientEmail}, "Kit Registration Confirmation", patientEmailContent)
	if err != nil {
		return fmt.Errorf("failed to send patient notification email: %v", err)
	}

	return nil
}
