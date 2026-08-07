package controllers

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// StatusHistory type for handling JSON data
type StatusHistory []struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// Scan implements the sql.Scanner interface
func (sh *StatusHistory) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSON value")
	}

	return json.Unmarshal(bytes, &sh)
}

// Value implements the driver.Valuer interface
func (sh StatusHistory) Value() (driver.Value, error) {
	return json.Marshal(sh)
}

type JoinedData struct {
	// Barcode Details
	BarcodeNumber    string    `json:"barcode_number"`
	BarcodeID        uint      `json:"barcode_id"`
	BarcodeCreatedAt time.Time `json:"barcode_created_at"`

	// Customer Details
	CustomerID        uint      `json:"customer_id"`
	Type              string    `json:"type"`
	FirstName         string    `json:"first_name"`
	LastName          string    `json:"last_name"`
	Email             string    `json:"email"`
	PhoneNumber       string    `json:"phone_number"`
	Country           string    `json:"country"`
	StreetAddress     string    `json:"street_address"`
	TownCity          string    `json:"town_city"`
	Region            string    `json:"region"`
	Postcode          string    `json:"postcode"`
	ShippingCountry   string    `json:"shipping_country"`
	ShippingAddress   string    `json:"shipping_address"`
	ShippingTownCity  string    `json:"shipping_town_city"`
	ShippingRegion    string    `json:"shipping_region"`
	ShippingPostcode  string    `json:"shipping_postcode"`
	ClinicID          string    `json:"clinic_id"`
	CustomerCreatedAt time.Time `json:"customer_created_at"`

	// Order Details
	OrderID            uint          `json:"order_id"`
	OrderNumber        string        `json:"order_number"`
	ProductName        string        `json:"product_name"`
	ProductDescription string        `json:"product_description"`
	ProductImage       string        `json:"product_image"`
	ProductPrice       float64       `json:"product_price"`
	ProductGstPrice    float64       `json:"product_gst_price"`
	ProductDiscount    float64       `json:"product_discount"`
	Quantity           int           `json:"quantity"`
	TotalPrice         float64       `json:"total_price"`
	PaymentStatus      string        `json:"payment_status"`
	OrderStatus        string        `json:"order_status"`
	OrderStatusHistory StatusHistory `json:"order_status_history"`
	TrackingID         string        `json:"tracking_id"`
	OrderCreatedAt     time.Time     `json:"order_created_at"`

	// Payment Details
	PaymentID        uint      `json:"payment_id"`
	TransactionID    string    `json:"transaction_id"`
	PaymentCreatedAt time.Time `json:"payment_created_at"`

	// Invoice Details
	InvoiceID        string    `json:"invoice_id"`
	InvoiceLink      string    `json:"invoice_link"`
	InvoiceCreatedAt time.Time `json:"invoice_created_at"`

	// Kit Registration Details
	KitRegistrationID        uint      `json:"kit_registration_id"`
	PatientFirstName         string    `json:"patient_first_name"`
	PatientLastName          string    `json:"patient_last_name"`
	PatientGender            string    `json:"patient_gender"`
	PatientAge               string    `json:"patient_age"`
	PatientEmail             string    `json:"patient_email"`
	KitStatus                string    `json:"kit_status"`
	Reason                   string    `json:"reason"`
	FilePath                 string    `json:"file_path"`
	KitRegistrationCreatedAt time.Time `json:"kit_registration_created_at"`

	// other details
	BarcodeCount    int  `json:"barcode_count"`
	QuantityMatches bool `json:"quantity_matches"`
}

func FetchBarcodeDetails(c *gin.Context) {
	// Get barcode number from URL parameters
	barcodeNumber := c.Param("id")

	// Validate the barcode number
	if len(barcodeNumber) != 30 {
		utils.JSONResponse(c, http.StatusBadRequest, "QRCode/Barcode number must be 30 characters long", nil)
		return
	}

	// Check if barcode contains only alphanumeric characters
	matched, err := regexp.MatchString(`^[a-zA-Z0-9]+$`, barcodeNumber)
	if err != nil || !matched {
		utils.JSONResponse(c, http.StatusBadRequest, "QRCode/Barcode number must contain only alphanumeric characters", nil)
		return
	}

	var joinedData JoinedData
	var barcodeCount int64

	result := config.DB.Debug().
		Model(&models.Barcode{}).
		Select(`
			barcodes.barcode_number,
			barcodes.id AS barcode_id,
			barcodes.created_at AS barcode_created_at,

			customers.id AS customer_id,
			customers.type,
			customers.first_name,
			customers.last_name,
			customers.email,
			customers.phone_number,
			customers.country,
			customers.street_address,
			customers.town_city,
			customers.region,
			customers.postcode,
			customers.shipping_country,
			customers.shipping_address,
			customers.shipping_town_city,
			customers.shipping_region,
			customers.shipping_postcode,
			customers.clinic_id,
			customers.created_at AS customer_created_at,

			orders.id AS order_id,
			orders.order_number,
			orders.product_name,
			orders.product_description,
			orders.product_image,
			orders.product_price,
			orders.product_gst_price,
			orders.product_discount,
			orders.quantity,
			orders.total_price,
			orders.payment_status,
			orders.order_status,
			orders.order_status_history::json AS order_status_history,
			orders.tracking_id,
			orders.created_at AS order_created_at,

			payments.id AS payment_id,
			payments.transaction_id,
			payments.created_at AS payment_created_at,

			invoices.invoice_id,
			invoices.invoice_link,
			invoices.created_at AS invoice_created_at,

			kit_registrations.id AS kit_registration_id,
			kit_registrations.patient_first_name,
			kit_registrations.patient_last_name,
			kit_registrations.patient_email,
			kit_registrations.patient_gender,
			kit_registrations.patient_age,
			kit_registrations.kit_status,
			kit_registrations.reason,
			kit_registrations.file_path,
			kit_registrations.created_at AS kit_registration_created_at
		`).
		Joins("LEFT JOIN orders ON orders.id = barcodes.order_id").
		Joins("LEFT JOIN customers ON customers.id = orders.customer_id").
		Joins("LEFT JOIN payments ON payments.order_id = orders.id").
		Joins("LEFT JOIN invoices ON invoices.payment_id = payments.id").
		Joins("LEFT JOIN kit_registrations ON kit_registrations.kit_barcode_order_id = barcodes.id").
		Where("barcodes.barcode_number = ? AND barcodes.is_deleted = ?", barcodeNumber, false).
		Where("(orders.is_deleted = ? OR orders.id IS NULL)", false).
		Where("(customers.is_deleted = ? OR customers.id IS NULL)", false).
		Where("(payments.is_deleted = ? OR payments.id IS NULL)", false).
		Where("(invoices.is_deleted = ? OR invoices.id IS NULL)", false).
		Where("(kit_registrations.is_deleted = ? OR kit_registrations.id IS NULL)", false).
		Scan(&joinedData)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusNotFound, "QRCode/Barcode  related data not found", nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, "Error fetching QRCode/Barcode details", nil)
		return
	}

	// Check if any data was found
	if joinedData.BarcodeID == 0 {
		utils.JSONResponse(c, http.StatusNotFound, "QRCode/Barcode not found", nil)
		return
	}

	// Fetch the count of barcodes for the related order
	err = config.DB.Debug().
		Model(&models.Barcode{}).
		Where("order_id = ? AND is_deleted = ?", joinedData.OrderID, false).
		Count(&barcodeCount).Error

	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Error fetching QRCode/Barcode count", nil)
		return
	}

	// Add the barcode count to the response
	joinedData.BarcodeCount = int(barcodeCount)
	joinedData.QuantityMatches = barcodeCount == int64(joinedData.Quantity)

	utils.JSONResponse(c, http.StatusOK, "Data fetched successfully", joinedData)
}
