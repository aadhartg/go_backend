// controllers/kit_registration_controller.go

package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

// VerifyBarcodeRequest represents the expected request body
type VerifyBarcodeRequest struct {
	BarcodeNumber string `json:"barcode_number" form:"barcode_number"`
}

// BarcodeVerificationResponse defines a custom response structure
type BarcodeVerificationResponse struct {
	// Barcode Details
	BarcodeNumber string `json:"barcode_number"`
	BarcodeID     uint   `json:"barcode_id"`

	// Order Details
	OrderID            uint    `json:"order_id"`
	OrderNumber        string  `json:"order_number"`
	ProductName        string  `json:"product_name"`
	ProductDescription string  `json:"product_description"`
	ProductImage       string  `json:"product_image"`
	ProductPrice       float64 `json:"product_price"`
	ProductGstPrice    float64 `json:"product_gst_price"`
	ProductDiscount    float64 `json:"product_discount"`
	Quantity           int     `json:"quantity"`

	// Customer Details
	CustomerID  uint   `json:"customer_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	ClinicID    string `json:"clinic_id"`

	// Address Details
	Country       string `json:"country"`
	StreetAddress string `json:"street_address"`
	TownCity      string `json:"town_city"`
	Region        string `json:"region"`
	Postcode      string `json:"postcode"`
}

type KiTRegisterRequest struct {
	PatientData string `json:"patient_data" form:"patient_data"`
}

type PatientDetail struct {
	FirstName      string `json:"first_name" validate:"required" form:"first_name"`
	LastName       string `json:"last_name" form:"last_name"`
	Email          string `json:"email" form:"email"`
	Gender         string `json:"gender" form:"gender"`
	Age            int    `json:"age" validate:"required,gt=0" form:"age"`
	OrderID        uint   `json:"order_id" form:"order_id"`
	CustomerID     uint   `json:"customer_id" form:"customer_id"`
	BarcodeNumber  string `json:"barcode_number" form:"barcode_number"`
	IsClinicInform bool   `json:"is_clinic_inform" form:"is_clinic_inform"`
}

// VerifyBarcode verifies a barcode and returns associated data.
func VerifyBarcode(c *gin.Context) {
	var req VerifyBarcodeRequest
	// Define allowed fields for this request
	allowedFields := []string{"barcode_number"}

	// Use the common request parser for both JSON and form data, and validate allowed fields
	err := utils.ParseRequestBody(c.Request, &req, allowedFields)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate input
	if req.BarcodeNumber == "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgBarcodeNumberRequired, nil)
		return
	}

	// Trim spaces and convert barcode to lowercase
	req.BarcodeNumber = strings.TrimSpace(req.BarcodeNumber)

	// Check length
	if len(req.BarcodeNumber) != 30 {
		utils.JSONResponse(c, http.StatusBadRequest,
			fmt.Sprintf("Invalid QRCode/Barcode length for %s: must be 30 characters", req.BarcodeNumber), nil)
		return
	}

	// Validate format (either alphabetic or numeric, no spaces)
	isValidBarcode := regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(req.BarcodeNumber) || regexp.MustCompile(`^\d+$`).MatchString(req.BarcodeNumber)

	if !isValidBarcode {
		utils.JSONResponse(c, http.StatusBadRequest,
			fmt.Sprintf("Invalid QRCode/Barcode format for %s: must be either alphabetic or numeric, and no spaces", req.BarcodeNumber), nil)
		return
	}

	// Begin transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to start database transaction", nil)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Unexpected error occurred", nil)
		}
	}()

	// Prepare the response
	var response BarcodeVerificationResponse

	// Fetch barcode data with associated order and customer details
	result := tx.Debug().
		Model(&models.Barcode{}).
		Select(`
			barcodes.barcode_number,
			barcodes.id AS barcode_id, 
			orders.id  AS order_id, 
			orders.order_number, 
			orders.product_name, 
			orders.product_description, 
			orders.product_image, 
			orders.product_price, 
			orders.product_gst_price, 
			orders.product_discount, 
			orders.quantity,
			customers.id AS customer_id,
			customers.first_name,
			customers.last_name,
			customers.email,
			customers.phone_number,
			customers.country,
			customers.street_address,
			customers.town_city,
			customers.region,
			customers.clinic_id,
			customers.postcode
		`).
		Joins("JOIN orders ON orders.id = barcodes.order_id").
		Where("orders.order_status = ? AND orders.is_deleted = ?", "Dispatched", false).
		Joins("JOIN customers ON customers.id = orders.customer_id").
		Where("barcodes.barcode_number = ? AND barcodes.is_deleted = ?", req.BarcodeNumber, false).
		Scan(&response)

	if result.Error != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusNotFound, "No QRCode/Barcode found", nil)
		return
	}

	// Check if kit registration already exists
	exists, err := services.IsKitRegistrationExists(tx, response.BarcodeID, response.OrderID, response.CustomerID)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	if exists {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgKitRegistrationExists, nil)
		return
	}

	// Commit transaction if all operations succeed
	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to commit transaction", nil)
		return
	}

	// Return the response
	utils.JSONResponse(c, http.StatusOK, "QRCode/Barcode verified successfully", response)
}

func KitRegister(c *gin.Context) {
	// Start a transaction
	tx := config.DB.Begin()

	// Defer rollback for unexpected panics
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var req KiTRegisterRequest
	allowedFields := []string{"patient_data"}

	// Parse and validate request body
	err := utils.ParseRequestBody(c.Request, &req, allowedFields)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if req.PatientData == "" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgPatientDataRequired, nil)
		return
	}

	// Decrypt patient data
	req.PatientData = strings.TrimSpace(req.PatientData)
	decryptedData, err := utils.Decrypt(req.PatientData)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgFailedToDecryptData, map[string]string{"error": err.Error()})
		return
	}

	var patientData PatientDetail
	if err := json.Unmarshal([]byte(decryptedData), &patientData); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPatientDataFormat, map[string]string{"error": err.Error()})
		return
	}

	patientData.FirstName = strings.TrimSpace(patientData.FirstName)
	patientData.LastName = strings.TrimSpace(patientData.LastName)
	patientData.Gender = strings.TrimSpace(strings.ToLower(patientData.Gender))
	patientData.Email = strings.TrimSpace(patientData.Email)

	// Validate patient data
	if err := services.ValidatePatientRequest(patientData.FirstName, patientData.LastName, patientData.Gender, patientData.Age, patientData.Email); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate barcode and order
	barcodeID, err := services.ValidateBarcodeAndOrder(tx, patientData.BarcodeNumber, patientData.OrderID)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate customer and retrieve data
	customer, err := services.ValidateCustomerID(tx, patientData.CustomerID)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate order and retrieve data
	order, err := services.ValidateOrderID(tx, patientData.OrderID)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Check for duplicate registration
	exists, err := services.IsKitRegistrationExists(tx, barcodeID, patientData.OrderID, patientData.CustomerID)
	if err != nil || exists {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgKitRegistrationExists, nil)
		return
	}

	// Insert registration
	err = services.InsertKitRegistration(tx, patientData.FirstName, patientData.LastName, patientData.Email, patientData.Gender, patientData.Age, barcodeID, patientData.CustomerID, patientData.OrderID, patientData.IsClinicInform)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Send email notifications
	emailErr := services.SendKitRegistrationNotificationEmails(
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
		order.ProductDescription,
		order.Quantity,
		order.ProductPrice,
		order.ProductGstPrice,
		order.ProductDiscount,
		order.TotalPrice,
		order.TrackingID,
		patientData.BarcodeNumber,
		patientData.FirstName,
		patientData.LastName,
		patientData.Gender,
		patientData.Email,
		patientData.Age, patientData.IsClinicInform)
	if emailErr != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, emailErr.Error(), nil)
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, utils.MsgPatientDataStoredSuccessfully, nil)
}
