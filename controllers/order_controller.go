// controllers/order_controller.go

package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type OrderRequest struct {
	Type               string  `json:"type" form:"type"`
	FirstName          string  `json:"first_name" form:"first_name" validate:"required,max=50,min=3"`
	LastName           string  `json:"last_name" form:"last_name" validate:"omitempty,max=50,min=3"`
	Email              string  `json:"email" form:"email" validate:"required,email,max=100"`
	PhoneNumber        string  `json:"phone_number" form:"phone_number" validate:"required,max=15,min=10"`
	Country            string  `json:"country" form:"country" validate:"required,max=50,min=3"`
	StreetAddress      string  `json:"street_address" form:"street_address" validate:"required,max=255,min=5"`
	TownCity           string  `json:"town_city" form:"town_city" validate:"required,max=100,min=5"`
	Region             string  `json:"region" form:"region" validate:"required,max=100,min=3"`
	Postcode           string  `json:"postcode" form:"postcode" validate:"required,max=20,min=3"`
	ShippingCountry    string  `json:"shipping_country" form:"shipping_country" validate:"required,max=50,min=3"`
	ShippingAddress    string  `json:"shipping_address" form:"shipping_address" validate:"required,max=255,min=5"`
	ShippingTownCity   string  `json:"shipping_town_city" form:"shipping_town_city" validate:"required,max=100,min=5"`
	ShippingRegion     string  `json:"shipping_region" form:"shipping_region" validate:"required,max=100,min=3"`
	ShippingPostcode   string  `json:"shipping_postcode" form:"shipping_postcode" validate:"required,max=20,min=3"`
	ProductName        string  `json:"product_name" form:"product_name" validate:"required,max=100,min=3"`
	ProductDescription string  `json:"product_description" form:"product_description" validate:"omitempty"`
	ProductImage       string  `json:"product_image" form:"product_image" validate:"omitempty,base64"`
	ProductPrice       float64 `json:"product_price" form:"product_price" validate:"required,numeric,gt=0"`
	ProductGstPrice    float64 `json:"product_gst_price" form:"product_gst_price" validate:"required,numeric,gt=0"`
	Quantity           int     `json:"quantity" form:"quantity" validate:"required,numeric,min=1"`
	ClinicID           string  `json:"clinic_id" form:"clinic_id" validate:"required,max=100,min=3"`
}

type PaymentResponse struct {
	PaymentURL string `json:"payment_url"`
}

// Define a new response structure for payment status
type PaymentStatusResponse struct {
	Action string `json:"action"` // success, cancel, or error
	Status string `json:"status"` // completed, failed, cancelled, processing
}

// OrderCreateHandler processes the complete order flow
func OrderCreateHandler(c *gin.Context) {
	var req OrderRequest
	allowedFields := []string{"first_name", "last_name", "email", "phone_number", "country",
		"street_address", "town_city", "region", "postcode", "product_name",
		"product_description", "product_image", "product_price", "quantity", "product_gst_price", "type", "shipping_country", "shipping_address", "shipping_town_city", "shipping_postcode", "shipping_region", "clinic_id"}

	if err := utils.ParseRequestBody(c.Request, &req, allowedFields); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	// Trim any spaces
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = strings.TrimSpace(req.Email)
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	req.Country = strings.TrimSpace(req.Country)
	req.StreetAddress = strings.TrimSpace(req.StreetAddress)
	req.TownCity = strings.TrimSpace(req.TownCity)
	req.Region = strings.TrimSpace(req.Region)
	req.Postcode = strings.TrimSpace(req.Postcode)
	req.ProductName = strings.TrimSpace(req.ProductName)
	req.ProductDescription = strings.TrimSpace(req.ProductDescription)
	req.ProductImage = strings.TrimSpace(req.ProductImage)
	req.Type = strings.TrimSpace(strings.ToLower(req.Type))
	req.ShippingCountry = strings.TrimSpace(req.ShippingCountry)
	req.ShippingAddress = strings.TrimSpace(req.ShippingAddress)
	req.ShippingTownCity = strings.TrimSpace(req.ShippingTownCity)
	req.ShippingRegion = strings.TrimSpace(req.ShippingRegion)
	req.ShippingPostcode = strings.TrimSpace(req.ShippingPostcode)
	req.ClinicID = strings.TrimSpace(req.ClinicID)

	if err := validateOrderRequest(&req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToStartTransaction, nil)
		return
	}
	defer tx.Rollback()

	// Check the order quantity
	if err := validateOrderQuantity(tx, &req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// 2. Process customer
	customer, err := processCustomer(tx, &req)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf(utils.MsgFailedToProcessCustomer, err.Error()), nil)
		return
	}

	// 3. Create order
	order, err := processOrderDetails(tx, customer, &req)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	// 4. Initialize PayPal payment
	paymentURL, _, err := initializePayPalPayment(tx, order, customer)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, fmt.Sprintf(utils.MsgFailedToInitializePayment, err.Error()), nil)
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCommitTransaction, nil)
		return
	}

	utils.JSONResponse(c, http.StatusOK, utils.MsgOrderCreatedSuccessfully, PaymentResponse{
		PaymentURL: paymentURL,
	})
}

// validateOrderQuantity checks if the requested order quantity is valid.
func validateOrderQuantity(tx *gorm.DB, req *OrderRequest) error {

	// Fetch the total available quantity of saliva kits.
	var availableQuantity int
	err := tx.Table("kits").
		Select("COALESCE(SUM(quantity), 0)").
		Where("type = ? AND is_deleted = ?", "saliva", false).
		Scan(&availableQuantity).Error
	if err != nil {
		return fmt.Errorf("Failed to fetch available saliva kit quantity: %w", err)
	}

	// Fetch the total quantity of saliva kits already ordered with completed payments.
	var totalOrderedQuantity int
	err = tx.Table("orders").
		Select("COALESCE(SUM(quantity), 0)").
		Where("payment_status = ? AND is_deleted = ?", "Completed", false).
		Scan(&totalOrderedQuantity).Error
	if err != nil {
		return fmt.Errorf("Failed to fetch total ordered quantity: %w", err)
	}

	// // Check if the requested quantity exceeds the remaining available quantity.
	// if req.Quantity > availableQuantity-totalOrderedQuantity {
	// 	return fmt.Errorf("Requested quantity exceeds available stock")
	// }

	return nil
}

// Additional validation functions
func validateOrderRequest(req *OrderRequest) error {
	// Existing validations
	if !utils.IsValidFirstName(req.FirstName) {
		return fmt.Errorf(utils.MsgInvalidFirstName)
	}
	if req.LastName != "" && !utils.IsValidLastName(req.LastName) {
		return fmt.Errorf(utils.MsgInvalidLastName)
	}
	if !utils.IsValidEmail(req.Email) {
		return fmt.Errorf(utils.MsgInvalidEmailFormat)
	}
	if !utils.IsValidContactNumber(req.PhoneNumber) {
		return fmt.Errorf(utils.MsgInvalidPhoneNumber)
	}
	if !utils.IsValidCountry(req.Country) {
		return fmt.Errorf(utils.MsgInvalidCountryName)
	}
	if !utils.IsValidStreetAddress(req.StreetAddress) {
		return fmt.Errorf(utils.MsgInvalidStreetAddress)
	}
	if !utils.IsValidTownCity(req.TownCity) {
		return fmt.Errorf(utils.MsgInvalidTownCity)
	}
	if !utils.IsValidRegion(req.Region) {
		return fmt.Errorf(utils.MsgInvalidRegion)
	}
	if !utils.IsValidPostcode(req.Postcode) {
		return fmt.Errorf(utils.MsgInvalidPostcode)
	}
	if !utils.IsValidCountry(req.ShippingCountry) {
		return fmt.Errorf(utils.MsgInvalidShippingCountryName)
	}
	if !utils.IsValidStreetAddress(req.ShippingAddress) {
		return fmt.Errorf(utils.MsgInvalidShippingAddress)
	}
	if !utils.IsValidTownCity(req.ShippingTownCity) {
		return fmt.Errorf(utils.MsgInvalidShippingTownCity)
	}
	if !utils.IsValidRegion(req.ShippingRegion) {
		return fmt.Errorf(utils.MsgInvalidShippingRegion)
	}
	if !utils.IsValidPostcode(req.ShippingPostcode) {
		return fmt.Errorf(utils.MsgInvalidShippingPostcode)
	}
	// Product validations
	if !utils.IsValidProductName(req.ProductName) {
		return fmt.Errorf(utils.MsgInvalidProductName)
	}

	// Product description validation (optional)
	if req.ProductDescription != "" && len(req.ProductDescription) > 1000 {
		return fmt.Errorf(utils.MsgProductDescriptionTooLong)
	}

	// Product image validation (optional, base64)
	if req.ProductImage != "" && !(utils.IsValidBase64Image(req.ProductImage) || utils.IsValidImageURL(req.ProductImage)) {
		return fmt.Errorf(utils.MsgInvalidProductImage)
	}

	// Updated price validation
	if req.ProductPrice <= 0 {
		return fmt.Errorf(utils.MsgInvalidProductPrice)
	}

	// Updated price validation
	if req.ProductGstPrice <= 0 {
		return fmt.Errorf(utils.MsgInvalidProductGstPrice)
	}

	// Updated quantity validation
	if req.Quantity <= 0 {
		return fmt.Errorf(utils.MsgInvalidQuantity)
	}

	if req.Type != "clinic" && req.Type != "customer" {
		return fmt.Errorf(utils.MsgInvalidCustomerType)
	}
	if req.Type == "clinic" {
		if req.ClinicID == "" {
			return fmt.Errorf(utils.MsgClinicIDRequired)
		}
		if len(req.ClinicID) > 100 {
			return fmt.Errorf(utils.MsgClinicIDInvalidLength)
		}
	}

	if req.Type == "customer" && req.Quantity > 1 {
		return fmt.Errorf(utils.MsgCustomerNotBuyMoreThenOne)
	}

	if req.Type == "clinic" && req.Quantity < 25 {
		return fmt.Errorf(utils.MsgClinicInvalidQuantity)
	}

	return nil
}

func processCustomer(tx *gorm.DB, req *OrderRequest) (*models.Customer, error) {
	var customer models.Customer

	// Check if a customer with the given email exists
	result := tx.Where("email = ?", req.Email).First(&customer)

	if result.Error == gorm.ErrRecordNotFound {
		// No existing customer, create a new one
		customer = models.Customer{
			FirstName:        req.FirstName,
			LastName:         req.LastName,
			Email:            req.Email,
			PhoneNumber:      req.PhoneNumber,
			Country:          req.Country,
			StreetAddress:    req.StreetAddress,
			TownCity:         req.TownCity,
			Region:           req.Region,
			Postcode:         req.Postcode,
			Type:             req.Type,
			ShippingCountry:  req.ShippingCountry,
			ShippingAddress:  req.ShippingAddress,
			ShippingTownCity: req.ShippingTownCity,
			ShippingRegion:   req.ShippingRegion,
			ShippingPostcode: req.ShippingPostcode,
			ClinicID:         req.ClinicID,
		}

		if err := tx.Create(&customer).Error; err != nil {
			return nil, err
		}
	} else if result.Error != nil {
		return nil, result.Error
	} else {
		// Existing customer found
		if customer.ClinicID != "" {
			// Customer is associated with a clinic
			if req.ClinicID == "" {
				return nil, fmt.Errorf("This email is already registered for a clinic. Please provide a Clinic ID to continue.")
			}
			if customer.ClinicID != req.ClinicID {
				return nil, fmt.Errorf("This email is already registered with a different clinic. Please use a different email or provide the Clinic ID.")
			}
		} else {
			// Customer is not associated with a clinic
			if req.ClinicID != "" {
				return nil, fmt.Errorf("The email address is already registered. Please use a different email for clinic-related purposes")
			}
		}
	}

	return &customer, nil
}

func processOrderDetails(tx *gorm.DB, customer *models.Customer, req *OrderRequest) (*models.Order, error) {

	// Initialize order status history with the initial status
	initialStatus := []models.StatusEntry{
		{
			Status:    "Pending",
			Timestamp: time.Now(),
		},
	}

	// Convert the initial status to JSON
	initialStatusJSON, err := json.Marshal(initialStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initial status: %v", err)
	}

	// Subtract the GST price from the product price to get the base price
	baseProductPrice := req.ProductPrice - req.ProductGstPrice

	// Get the discount based on the order quantity
	discountPercentage, err := services.GetProductDiscount(tx, req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product discount: %v", err)
	}

	// Calculate the product discount
	productDiscount := (baseProductPrice * discountPercentage) / 100

	// Update the total price after applying the discount
	discountedPrice := baseProductPrice - productDiscount
	totalPrice := (discountedPrice + req.ProductGstPrice) * float64(req.Quantity)

	orderID, err := utils.GenerateUniqueOrderID(config.DB)
	if err != nil {
		return nil, fmt.Errorf("error generating unique order ID: %v", err)
	}

	order := models.Order{
		CustomerID:         customer.ID,
		ProductName:        req.ProductName,
		ProductDescription: req.ProductDescription,
		ProductImage:       req.ProductImage,
		ProductPrice:       baseProductPrice,
		ProductGstPrice:    req.ProductGstPrice,
		ProductDiscount:    discountPercentage,
		Quantity:           req.Quantity,
		OrderNumber:        orderID,
		TotalPrice:         totalPrice,
		PaymentStatus:      "Pending",
		OrderStatus:        "Pending",
		OrderStatusHistory: datatypes.JSON(initialStatusJSON),
	}

	if err := tx.Create(&order).Error; err != nil {
		return nil, err
	}

	return &order, nil
}

func initializePayPalPayment(tx *gorm.DB, order *models.Order, customer *models.Customer) (string, uint, error) {
	// Create payment record
	payment := &models.Payment{
		OrderID:         order.ID,
		PaymentStatus:   "Pending",
		Amount:          order.TotalPrice,
		ProductGstPrice: order.ProductGstPrice,
		ProductDiscount: order.ProductDiscount,
	}
	if err := tx.Create(payment).Error; err != nil {
		return "", 0, err
	}

	// Initialize PayPal payment
	accessToken, err := utils.GetPayPalAccessToken()
	if err != nil {
		return "", 0, err
	}

	paypalOrder, err := utils.CreatePayPalOrder(payment.ID, order.TotalPrice, accessToken, order, customer)
	if err != nil {
		log.Println(err)

		return "", 0, err
	}
	// Update payment with PayPal transaction ID
	payment.TransactionID = paypalOrder.ID
	if err := tx.Save(payment).Error; err != nil {
		return "", 0, err
	}

	// Get PayPal approval URL
	var approvalURL string
	for _, link := range paypalOrder.Links {
		if link.Rel == "approve" {
			approvalURL = link.Href
			break
		}
	}

	return approvalURL, payment.ID, nil
}

// HandlePaymentSuccess handles the payment success, cancel, or error actions
func HandlePaymentSuccess(c *gin.Context) {
	paymentID := c.Query("payment_id")
	paypalOrderID := c.Query("token")
	paymentAction := c.Query("action") // success, cancel, or error

	if paymentID == "" || paypalOrderID == "" || paymentAction == "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgMissingPaymentInformation, nil)
		return
	}

	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToStartTransactionAgain, nil)
		return
	}
	defer tx.Rollback()

	var err error
	var paymentStatus string
	switch paymentAction {
	case "success":
		err = handleSuccessfulPayment(tx, paymentID, paypalOrderID)
		paymentStatus = "completed"
		if err != nil {
			paymentStatus = "failed"
		}
	case "cancel":
		err = handleCancelledPayment(tx, paymentID)
		paymentStatus = "cancelled"

	default:
		err = handleFailedPayment(tx, paymentID)
		paymentStatus = "failed"

	}

	if err != nil {
		response := PaymentStatusResponse{
			Action: paymentAction,
			Status: paymentStatus,
		}
		utils.JSONResponse(c, http.StatusInternalServerError, err.Error(), response)
		return
	}

	if err := tx.Commit().Error; err != nil {
		response := PaymentStatusResponse{
			Action: paymentAction,
			Status: "failed",
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCompletePaymentProcess, response)
		return
	}
	response := PaymentStatusResponse{
		Action: paymentAction,
		Status: paymentStatus,
	}
	message := getResponseMessage(paymentAction)
	utils.JSONResponse(c, http.StatusOK, message, response)
}
func getResponseMessage(action string) string {
	switch action {
	case "success":
		return utils.MsgPaymentCompletedSuccessfully
	case "cancel":
		return "Payment was cancelled by the user"
	default:
		return "Payment processing failed"
	}
}

func handleSuccessfulPayment(tx *gorm.DB, paymentID, paypalOrderID string) error {
	// First verify and capture the payment
	if err := captureAndVerifyPayment(paypalOrderID); err != nil {
		// If capture fails, mark as failed and return error
		updatePaymentAndOrderStatus(tx, paymentID, utils.PaymentStatusFailed, utils.OrderStatusFailed)
		return fmt.Errorf(utils.MsgPaymentVerificationFailed, err.Error())
	}

	// Update statuses
	if err := updatePaymentAndOrderStatus(tx, paymentID, utils.PaymentStatusCompleted, utils.OrderStatusPending); err != nil {
		return err
	}

	// Generate invoice and send emails
	return handlePostPaymentProcessing(tx, paymentID)
}

func handleCancelledPayment(tx *gorm.DB, paymentID string) error {
	return updatePaymentAndOrderStatus(tx, paymentID, utils.PaymentStatusCancelled, utils.OrderStatusCancelled)
}

func handleFailedPayment(tx *gorm.DB, paymentID string) error {
	return updatePaymentAndOrderStatus(tx, paymentID, utils.PaymentStatusFailed, utils.OrderStatusFailed)
}

func updatePaymentAndOrderStatus(tx *gorm.DB, paymentID, paymentStatus, orderStatus string) error {
	var payment models.Payment
	if err := tx.First(&payment, paymentID).Error; err != nil {
		return fmt.Errorf(utils.MsgPaymentNotFound)
	}

	payment.PaymentStatus = paymentStatus
	if err := tx.Save(&payment).Error; err != nil {
		return fmt.Errorf(utils.MsgFailedToUpdatePayment)
	}

	var order models.Order
	if err := tx.First(&order, payment.OrderID).Error; err != nil {
		return fmt.Errorf(utils.MsgOrderNotFound)
	}

	// Retrieve existing status history
	var statusHistory []models.StatusEntry
	if len(order.OrderStatusHistory) > 0 {
		if err := json.Unmarshal(order.OrderStatusHistory, &statusHistory); err != nil {
			return fmt.Errorf("failed to unmarshal status history: %v", err)
		}
	}

	// Add new status entry
	newStatusEntry := models.StatusEntry{
		Status:    orderStatus,
		Timestamp: time.Now(),
	}
	statusHistory = append(statusHistory, newStatusEntry)

	// Convert updated status history to JSON
	updatedStatusJSON, err := json.Marshal(statusHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal status history: %v", err)
	}

	order.PaymentStatus = paymentStatus
	if orderStatus != utils.OrderStatusPending {

		order.OrderStatus = orderStatus
		order.OrderStatusHistory = datatypes.JSON(updatedStatusJSON)
	}
	return tx.Save(&order).Error
}

func captureAndVerifyPayment(paypalOrderID string) error {
	accessToken, err := utils.GetPayPalAccessToken()
	if err != nil {
		return err
	}
	return utils.CapturePayPalPayment(paypalOrderID, accessToken)
}

func handlePostPaymentProcessing(tx *gorm.DB, paymentID string) error {
	var (
		payment  models.Payment
		order    models.Order
		customer models.Customer
	)

	// Get all necessary data
	if err := tx.First(&payment, paymentID).Error; err != nil {
		return err
	}
	if err := tx.First(&order, payment.OrderID).Error; err != nil {
		return err
	}
	if err := tx.First(&customer, order.CustomerID).Error; err != nil {
		return err
	}

	invoiceUUID, err := utils.GenerateUniqueOrderID(config.DB)
	if err != nil {
		return fmt.Errorf("error generating unique order ID: %v", err)
	}

	// Generate invoice
	invoicePath, err := generateInvoice(&payment, &customer, invoiceUUID)
	if err != nil {
		return err
	}

	// Save invoice
	invoice := models.Invoice{
		PaymentID:       payment.ID,
		InvoiceLink:     invoicePath,
		Price:           payment.Amount,
		ProductGstPrice: payment.ProductGstPrice,
		ProductDiscount: payment.ProductDiscount,
		InvoiceID:       invoiceUUID,
	}
	if err := tx.Create(&invoice).Error; err != nil {
		return err
	}

	// Send confirmation emails
	if err := sendConfirmationEmails(&customer, &order, &invoice); err != nil {
		return err
	}

	notificationUsername := fmt.Sprintf(
		"%s %s",
		customer.FirstName, customer.LastName,
	)

	// Notify admins and super-admins about the new order
	message := fmt.Sprintf(
		"%s %s has placed a new order for %d units of %s, totaling $%.2f.",
		customer.FirstName,
		customer.LastName,
		order.Quantity,
		order.ProductName,
		payment.Amount,
	)

	metadata := map[string]interface{}{
		"status":       "Pending",
		"module":       "Manage Orders",
		"order_number": order.OrderNumber,
	}

	if err := services.NotifyUsersByRoles(
		tx, // Using the existing transaction
		nil,
		[]string{"super-admin", "admin"},
		"New Order Received",
		"Order Purchase",
		message,
		notificationUsername,
		"orders",
		order.ID,
		metadata,
	); err != nil {
		return fmt.Errorf("failed to send admin notifications: %v", err)
	}

	return nil
}

func generateInvoice(payment *models.Payment, customer *models.Customer, invoiceUUID string) (string, error) {
	// Get the order details for this payment
	var order models.Order
	if err := config.DB.First(&order, payment.OrderID).Error; err != nil {
		return "", err
	}

	// Precise calculations
	unitPrice := order.ProductPrice
	quantity := float64(order.Quantity)

	// Calculate subtotal
	subtotal := math.Round(unitPrice*quantity*100) / 100

	// Calculate discount
	discountRate := order.ProductDiscount / 100
	discountAmount := math.Round(subtotal*discountRate*100) / 100

	// Calculate GST
	gstPrice := order.ProductGstPrice
	gstRate := (gstPrice / unitPrice) * 100 // Calculate GST percentage
	totalGst := math.Round(gstPrice*quantity*100) / 100

	// Calculate total amount
	totalAmount := math.Round((subtotal+totalGst-discountAmount)*100) / 100

	// Initialize PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Set light grey background for the entire page
	pdf.SetFillColor(245, 245, 245)
	pdf.Rect(0, 0, 210, 297, "F")

	// Set default font
	pdf.SetFont("Arial", "", 10)

	// Add company logo centered at the top with reduced bottom margin
	logoPath := "public/images/invoicelogo.png"
	if _, err := os.Stat(logoPath); err == nil {
		pageWidth := 210.0
		imgWidth := 40.0
		xPos := (pageWidth - imgWidth) / 2
		pdf.Image(logoPath, xPos, 10, imgWidth, 0, false, "", 0, "")
	}

	// Add border closer to logo
	pdf.Line(10, 30, 200, 30)

	// Center aligned "TAX INVOICE" heading with increased bottom margin
	pdf.SetY(35)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(190, 10, "TAX INVOICE", "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Invoice information in two columns

	// Modified helper function for adding left column details with proper text wrapping
	addLeftDetail := func(label, value string, maxWidth float64) float64 {
		startY := pdf.GetY()
		pdf.SetX(10)
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(40, 5, label+":")

		// Calculate wrapped text height
		pdf.SetFont("Arial", "", 10)
		wrappedText := pdf.SplitText(value, maxWidth)
		textHeight := float64(len(wrappedText)) * 5

		// Reset Y position to write the value
		pdf.SetY(startY)
		pdf.SetX(50)

		for i, line := range wrappedText {
			if i > 0 {
				pdf.SetX(50)
			}
			pdf.Cell(maxWidth, 5, line)
			pdf.Ln(5)
		}

		return startY + textHeight
	}

	// Modified helper function for adding right column details with proper text wrapping
	addRightDetail := func(label, value string, maxWidth float64) float64 {
		startY := pdf.GetY()
		pdf.SetX(120)
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(40, 5, label+":")

		// Calculate wrapped text height
		pdf.SetFont("Arial", "", 10)
		wrappedText := pdf.SplitText(value, maxWidth)
		textHeight := float64(len(wrappedText)) * 5

		// Reset Y position to write the value
		pdf.SetY(startY)
		pdf.SetX(160)

		for i, line := range wrappedText {
			if i > 0 {
				pdf.SetX(160)
			}
			pdf.Cell(maxWidth, 5, line)
			pdf.Ln(5)
		}

		return startY + textHeight
	}

	// Start position for both columns
	startY := pdf.GetY()
	leftY := startY
	rightY := startY

	customerName := utils.CapitalizeWords(fmt.Sprintf("%s %s", customer.FirstName, customer.LastName))
	address := fmt.Sprintf("%s, %s, %s, %s, %s",
		customer.StreetAddress,
		customer.TownCity,
		customer.Region,
		customer.Country,
		customer.Postcode)
	shippingAddress := fmt.Sprintf("%s, %s, %s, %s, %s",
		customer.ShippingAddress,
		customer.ShippingTownCity,
		customer.ShippingRegion,
		customer.ShippingCountry,
		customer.ShippingPostcode)

	// Add left column details with spacing
	leftDetails := []struct {
		label string
		value string
	}{
		{"Invoice To", customerName},
		{"Customer Address", address},
		{"Shipping Address", shippingAddress},
		{"Customer Contact", customer.PhoneNumber},
		{"Invoice Number", fmt.Sprintf("#%s", invoiceUUID)},
		{"Order Number", fmt.Sprintf("#%s", order.OrderNumber)},
		{"Invoice Date", payment.CreatedAt.Format("02 Jan, 2006")},
		{"Due Date", payment.CreatedAt.Format("02 Jan, 2006")},
	}

	// Add left column details and track maximum Y position
	for _, detail := range leftDetails {
		pdf.SetY(leftY)
		newY := addLeftDetail(detail.label, detail.value, 60)
		leftY = newY + 2 // Add small spacing between rows
	}

	// Add clinic ID if provided
	if customer.ClinicID != "" {
		pdf.SetY(leftY)
		newY := addLeftDetail("Clinic ID", customer.ClinicID, 60)
		leftY = newY + 2
	}

	companyDetails := []struct {
		label string
		value string
	}{
		{"Company Name", "Theranostics Lab"},
		{"Company Address", "26 Volcanic St, Mt Eden, Auckland 1041"},
		{"Company GSTIN", "# 102-545-427"},
		{"Owner Name", "Dr Patrick Gladding"},
		{"Contact Number", "+64220424325"},
		{"Email", "patrickg@theranosticslab.com"},
		{"Website", "www.theranostics.co.nz"},
	}

	// Add right column details and track maximum Y position
	for _, detail := range companyDetails {
		pdf.SetY(rightY)
		newY := addRightDetail(detail.label, detail.value, 40)
		rightY = newY + 2
	}

	// Get the maximum Y position from both columns
	contentEndY := math.Max(leftY, rightY) + 10

	// Check if we need to add a new page for the table
	if contentEndY+100 > 297 { // Reserve 100mm for table and totals
		pdf.AddPage()
		contentEndY = 10
	}

	// Set the table's starting position dynamically
	tableStartY := contentEndY + 5

	// Horizontal line before table
	pdf.Line(10, tableStartY, 200, tableStartY)
	tableStartY += 5

	// Set position for table headers
	pdf.SetXY(10, tableStartY)
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetFont("Arial", "B", 10)

	// Table headers with right alignment for numeric columns
	pdf.Cell(60, 8, "ITEM DETAIL")
	pdf.SetX(70)
	pdf.CellFormat(30, 8, "QTY", "", 0, "R", false, 0, "")
	pdf.SetX(100)
	pdf.CellFormat(40, 8, "RATE", "", 0, "R", false, 0, "")
	pdf.SetX(140)
	pdf.CellFormat(60, 8, "AMOUNT", "", 0, "R", false, 0, "")

	// Bottom line for table header
	pdf.Line(10, tableStartY+8, 200, tableStartY+8)

	// Move to next row
	tableStartY += 8
	pdf.SetY(tableStartY)

	// Product name and details row
	pdf.SetX(10)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(60, 8, order.ProductName)
	pdf.SetFont("Arial", "", 10)
	pdf.SetX(70)
	pdf.CellFormat(30, 8, fmt.Sprintf("%d", order.Quantity), "", 0, "R", false, 0, "")
	pdf.SetX(100)
	pdf.CellFormat(40, 8, fmt.Sprintf("$%.2f", unitPrice), "", 0, "R", false, 0, "")
	pdf.SetX(140)
	pdf.CellFormat(60, 8, fmt.Sprintf("$%.2f", subtotal), "", 0, "R", false, 0, "")
	pdf.Ln(8)

	// Description with height calculation
	if order.ProductDescription != "" {
		descriptionY := pdf.GetY()
		pdf.SetFont("Arial", "", 8)
		description := order.ProductDescription
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		// Calculate description height
		descLines := pdf.SplitText(description, 60)
		descHeight := float64(len(descLines)) * 4

		// Check if we need a new page for description
		if descriptionY+descHeight+20 > 297 {
			pdf.AddPage()
			descriptionY = 10
			pdf.SetY(descriptionY)
		}

		pdf.SetX(10)
		pdf.MultiCell(60, 4, description, "", "L", false)
	}

	// Add horizontal line after table data
	currentY := pdf.GetY() + 5
	pdf.Line(10, currentY, 200, currentY)

	// Check if we need a new page for totals
	if currentY+50 > 297 { // Reserve 50mm for totals section
		pdf.AddPage()
		currentY = 10
	}

	// Move down for totals section
	currentY += 10
	pdf.SetY(currentY)
	// Helper function to add total row
	addTotalRow := func(label string, amount float64, isNegative bool) {
		pdf.SetX(140)
		pdf.Cell(30, 8, label)
		pdf.SetX(170)
		amountStr := fmt.Sprintf("$%.2f", amount)
		if isNegative {
			amountStr = "- " + amountStr
		}
		pdf.CellFormat(30, 8, amountStr, "", 0, "R", false, 0, "")
		pdf.Ln(8)
	}

	// Add total rows
	pdf.SetFont("Arial", "", 10)
	addTotalRow("Subtotal:", subtotal, false)
	addTotalRow(fmt.Sprintf("Discount(%.2f%%):", order.ProductDiscount), discountAmount, true)
	addTotalRow(fmt.Sprintf("GST(%.2f%%):", gstRate), totalGst, false)

	// Line before final total
	currentY = pdf.GetY()
	pdf.Line(140, currentY, 200, currentY)
	pdf.Ln(2)

	// Final total
	pdf.SetFont("Arial", "B", 10)
	addTotalRow("Total:", totalAmount, false)

	// Save PDF
	invoicePath := filepath.Join("public/files", fmt.Sprintf("invoice_%s.pdf", invoiceUUID))
	if err := os.MkdirAll(filepath.Dir(invoicePath), 0755); err != nil {
		return "", err
	}

	// Generate the PDF and save to file
	if err := pdf.OutputFileAndClose(invoicePath); err != nil {
		return "", err
	}

	return filepath.Join("", fmt.Sprintf("files/invoice_%s.pdf", invoiceUUID)), nil
}

func sendConfirmationEmails(customer *models.Customer, order *models.Order, invoice *models.Invoice) error {
	// Send customer email
	customerEmail := emails.CustomerOrderConfirmationEmail(
		order.OrderNumber,
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
		customer.ClinicID,
		order.ProductName,
		order.ProductDescription,
		order.Quantity,
		order.ProductPrice,
		order.ProductGstPrice,
		order.ProductDiscount,
		order.TotalPrice,
		order.ProductImage,
		invoice.InvoiceLink,
		config.AppConfig.AppUrl,
	)
	if err := config.SendEmail([]string{customer.Email}, "Order Confirmation", customerEmail); err != nil {
		return err
	}
	// Send admin notification if configured
	if adminEmail := config.AppConfig.ClientEmail; adminEmail != "" {

		// Create the email body for the admin notification
		adminEmailBody := emails.NewOrderNotificationEmail(
			order.OrderNumber,
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
			customer.ClinicID,
			order.ProductName,
			order.ProductDescription,
			order.Quantity,
			order.ProductPrice,
			order.ProductGstPrice,
			order.ProductDiscount,
			order.TotalPrice,
			order.ProductImage,
			invoice.InvoiceLink,
			config.AppConfig.AppUrl)

		// Send the email to the admin
		if err := config.SendEmail([]string{adminEmail}, "New Order Received", adminEmailBody); err != nil {
			return err
		}
	}

	return nil
}
