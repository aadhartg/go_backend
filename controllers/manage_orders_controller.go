// controllers/manage_orders_controller.go

package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"theransticslabs/m/config"
	"theransticslabs/m/emails"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/models"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Response structure for paginated list of customers with their orders
type OrdersFetchedDataResponse struct {
	Page          int    `json:"page"`
	PerPage       int    `json:"per_page"`
	Sort          string `json:"sort"`
	SortColumn    string `json:"sort_column"`
	SearchText    string `json:"search_text"`
	TotalRecords  int64  `json:"total_records"`
	PaymentStatus string `json:"payment_status"`
	OrderStatus   string `json:"order_status"`
	TotalPages    int    `json:"total_pages"`

	Records []OrdersResponse `json:"records"`
}

// Custom struct to scan additional fields
type OrderWithBarcodes struct {
	models.Order
	BarcodeCount   int64  `gorm:"column:barcode_count"`
	BarcodeNumbers string `gorm:"column:barcode_numbers"`
}

// OrdersResponse defines the structure for the API response
type OrdersResponse struct {
	ID                 uint                   `json:"id"`
	ProductName        string                 `json:"product_name"`
	ProductDescription string                 `json:"product_description"`
	ProductImage       string                 `json:"product_image"`
	ProductPrice       float64                `json:"product_price"`
	ProductGstPrice    float64                `json:"product_gst_price"`
	ProductDiscount    float64                `json:"product_discount"`
	Quantity           int                    `json:"quantity"`
	OrderNumber        string                 `json:"order_number"`
	TotalPrice         float64                `json:"total_price"`
	PaymentStatus      string                 `json:"payment_status"`
	OrderStatus        string                 `json:"order_status"`
	TrackingID         string                 `json:"tracking_id"`
	CreatedAt          time.Time              `json:"created_at"`
	Customer           OrderCustomerResponse  `json:"customer"`
	BarcodeCount       int64                  `json:"barcode_count"`
	BarcodeNumbers     string                 `json:"barcode_numbers"`
	Payments           []OrderPaymentResponse `json:"payments,omitempty"`
}

// Customer represents the essential customer information
type OrderCustomerResponse struct {
	ID               uint      `json:"id"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Email            string    `json:"email"`
	PhoneNumber      string    `json:"phone_number"`
	Country          string    `json:"country"`
	StreetAddress    string    `json:"street_address"`
	TownCity         string    `json:"town_city"`
	Region           string    `json:"region"`
	Postcode         string    `json:"postcode"`
	ClinicID         string    `json:"clinic_id"`
	CreatedAt        time.Time `json:"created_at"`
	ShippingCountry  string    `json:"shipping_country"`
	ShippingAddress  string    `json:"shipping_address"`
	ShippingTownCity string    `json:"shipping_town_city"`
	ShippingRegion   string    `json:"shipping_region"`
	ShippingPostcode string    `json:"shipping_postcode"`
}

// Payment represents the payment information with related invoices
type OrderPaymentResponse struct {
	ID              uint                  `json:"id"`
	PaymentStatus   string                `json:"payment_status"`
	TransactionID   string                `json:"transaction_id"`
	Amount          float64               `json:"amount"`
	ProductGstPrice float64               `json:"product_gst_price"`
	ProductDiscount float64               `json:"product_discount"`
	CreatedAt       time.Time             `json:"created_at"`
	Invoices        []OrderPaymentInvoice `json:"invoices,omitempty"`
}

// Invoice represents the invoice information
type OrderPaymentInvoice struct {
	ID              uint    `json:"id"`
	InvoiceLink     string  `json:"invoice_link"`
	Price           float64 `json:"price"`
	ProductGstPrice float64 `json:"product_gst_price"`
	ProductDiscount float64 `json:"product_discount"`
	InvoiceID       string  `json:"invoice_id"`
}

// CreateAssignKitRequest represents the request structure for assigning multiple barcodes
type CreateAssignKitRequest struct {
	BarcodeNumbers string `json:"barcode_numbers" form:"barcode_numbers" validate:"required"`
}

// CreateAssignKitRequest represents the request body for assigning a kit.
type UpdateOrderStatusRequest struct {
	Status     string `json:"status" form:"status" validate:"required"`
	TrackingID string `json:"tracking_id" form:"tracking_id" validate:"required"`
}

// OrderCountResponse defines the structure of the response
type OrderCountResponse struct {
	Dispatched    int64 `json:"dispatched"`
	Processing    int64 `json:"processing"`
	Pending       int64 `json:"pending"`
	Shipped       int64 `json:"shipped"`
	TotalQuantity int64 `json:"total_quantity"`
}

// Allowed order statuses
var allowedStatuses = []string{"Pending", "Processing", "Shipped", "Delivered", "Cancelled", "Dispatched"}

func GetOrdersWithCustomersInvoicePayments(c *gin.Context) {
	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "payment_status", "order_status"}

	// Parse query parameters with default values
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidQueryParameters, nil)
		return
	}

	// Default and validation for 'page'
	page := 1
	if val := query.Get("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPageParameter, nil)
			return
		}
	}

	// Default and validation for 'per_page'
	perPage := 10
	if val := query.Get("per_page"); val != "" {
		if pp, err := strconv.Atoi(val); err == nil && pp > 0 {
			perPage = pp
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPerPageParameter, nil)
			return
		}
	}

	// Default and validation for 'sort'
	sort := "desc"
	if val := strings.ToLower(query.Get("sort")); val == "asc" || val == "desc" {
		sort = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortParameter, nil)
		return
	}

	// Default and validation for 'sort_column'
	sortColumn := "created_at"
	validSortColumns := []string{"id", "product_name", "product_price", "product_gst_price", "quantity", "total_price", "payment_status", "order_status", "created_at", "customer_first_name", "customer_last_name", "customer_clinic_id", "customer_email", "customer_phone_number", "order_number"}
	if val := strings.ToLower(query.Get("sort_column")); val != "" {
		if utils.StringInSlice(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortColumnParameter, nil)
			return
		}
	}

	// Optional 'search_text'
	searchText := strings.TrimSpace(query.Get("search_text"))

	// Default and validation for 'payment_status'
	paymentStatus := "completed"
	if val := strings.ToLower(query.Get("payment_status")); val == "pending" || val == "completed" || val == "failed" || val == "all" {
		paymentStatus = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPaymentStatus, nil)
		return
	}

	// Default and validation for 'order_status'
	orderStatus := "all"
	if val := strings.ToLower(query.Get("order_status")); val == "pending" || val == "shipped" || val == "delivered" || val == "cancelled" || val == "processing" || val == "dispatched" || val == "all" {
		orderStatus = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidOrderStatus, nil)
		return
	}

	// Initialize database query with explicit joins
	ordersDB := config.DB.Model(&models.Order{}).
		Joins("LEFT JOIN customers ON orders.customer_id = customers.id").
		Joins("LEFT JOIN (SELECT order_id, COUNT(*) as barcode_count, STRING_AGG(barcode_number, ', ') as barcode_numbers FROM barcodes WHERE is_deleted = false GROUP BY order_id) bc ON orders.id = bc.order_id").
		Preload("Customer").
		Preload("Payments", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_deleted = ?", false)
		}).
		Preload("Payments.Invoices", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_deleted = ?", false)
		}).
		Where("orders.is_deleted = ?", false)

	// Apply search filter if provided
	if searchText != "" {
		searchPattern := "%" + searchText + "%"
		ordersDB = ordersDB.Where(
			`orders.product_name ILIKE ? OR 
				CAST(orders.order_number AS TEXT) ILIKE ? OR 
				CAST(orders.product_price AS TEXT) ILIKE ? OR 
				CAST(orders.product_gst_price AS TEXT) ILIKE ? OR 
				CAST(orders.quantity AS TEXT) ILIKE ? OR 
				CAST(orders.total_price AS TEXT) ILIKE ? OR 
				orders.payment_status ILIKE ? OR 
				orders.order_status ILIKE ? OR 
				TO_CHAR(orders.created_at, 'YYYY-MM-DD') ILIKE ? OR
				customers.first_name ILIKE ? OR 
				customers.last_name ILIKE ? OR 
				customers.email ILIKE ? OR 
				customers.clinic_id ILIKE ? OR 
				customers.phone_number ILIKE ? OR
				bc.barcode_numbers ILIKE ?`,
			searchPattern, searchPattern, searchPattern, searchPattern,
			searchPattern, searchPattern, searchPattern, searchPattern,
			searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Apply payment status filter
	if paymentStatus != "all" {
		ordersDB = ordersDB.Where("LOWER(orders.payment_status) = ?", strings.ToLower(paymentStatus))
	}

	// Apply order status filter
	if orderStatus != "all" {
		ordersDB = ordersDB.Where("LOWER(orders.order_status) = ?", strings.ToLower(orderStatus))
	}

	// Get total records count
	var totalRecords int64
	if err := ordersDB.Count(&totalRecords).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCountRecords, nil)
		return
	}

	// Calculate total pages
	totalPages := int((totalRecords + int64(perPage) - 1) / int64(perPage))
	if totalRecords == 0 {
		totalPages = 0
	}

	// Handle sorting
	var orderQuery string
	switch sortColumn {
	case "customer_first_name":
		orderQuery = "customers.first_name " + sort
	case "customer_last_name":
		orderQuery = "customers.last_name " + sort
	case "customer_email":
		orderQuery = "customers.email " + sort
	case "customer_phone_number":
		orderQuery = "customers.phone_number " + sort
	case "customer_clinic_id":
		orderQuery = "customers.clinic_id " + sort
	default:
		orderQuery = "orders." + sortColumn + " " + sort
	}

	// Apply default sorting (Pending orders first)
	ordersDB = ordersDB.Order("CASE WHEN orders.order_status = 'Pending' THEN 0 ELSE 1 END").
		Order(orderQuery)

	// Apply pagination
	offset := (page - 1) * perPage
	var orders []OrderWithBarcodes
	if err := ordersDB.
		Select("orders.*, COALESCE(bc.barcode_count, 0) as barcode_count, COALESCE(bc.barcode_numbers, '') as barcode_numbers").
		Limit(perPage).
		Offset(offset).
		Find(&orders).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Transform and build response
	var responseOrders []OrdersResponse
	for _, dbOrder := range orders {
		customer := OrderCustomerResponse{
			ID:               dbOrder.Customer.ID,
			FirstName:        dbOrder.Customer.FirstName,
			LastName:         dbOrder.Customer.LastName,
			Email:            dbOrder.Customer.Email,
			PhoneNumber:      dbOrder.Customer.PhoneNumber,
			Country:          dbOrder.Customer.Country,
			StreetAddress:    dbOrder.Customer.StreetAddress,
			TownCity:         dbOrder.Customer.TownCity,
			Region:           dbOrder.Customer.Region,
			Postcode:         dbOrder.Customer.Postcode,
			ClinicID:         dbOrder.Customer.ClinicID,
			ShippingCountry:  dbOrder.Customer.ShippingCountry,
			ShippingAddress:  dbOrder.Customer.ShippingAddress,
			ShippingTownCity: dbOrder.Customer.ShippingTownCity,
			ShippingRegion:   dbOrder.Customer.ShippingRegion,
			ShippingPostcode: dbOrder.Customer.ShippingPostcode,
		}

		var payments []OrderPaymentResponse
		for _, dbPayment := range dbOrder.Payments {
			var invoices []OrderPaymentInvoice
			for _, dbInvoice := range dbPayment.Invoices {
				invoices = append(invoices, OrderPaymentInvoice{
					ID:              dbInvoice.ID,
					InvoiceLink:     dbInvoice.InvoiceLink,
					Price:           dbInvoice.Price,
					ProductGstPrice: dbInvoice.ProductGstPrice,
					InvoiceID:       dbInvoice.InvoiceID,
				})
			}
			payments = append(payments, OrderPaymentResponse{
				ID:              dbPayment.ID,
				PaymentStatus:   dbPayment.PaymentStatus,
				TransactionID:   dbPayment.TransactionID,
				Amount:          dbPayment.Amount,
				ProductGstPrice: dbPayment.ProductGstPrice,
				ProductDiscount: dbPayment.ProductDiscount,
				CreatedAt:       dbPayment.CreatedAt,
				Invoices:        invoices,
			})
		}

		responseOrders = append(responseOrders, OrdersResponse{
			ID:                 dbOrder.ID,
			ProductName:        dbOrder.ProductName,
			ProductDescription: dbOrder.ProductDescription,
			ProductImage:       dbOrder.ProductImage,
			ProductPrice:       dbOrder.ProductPrice,
			ProductGstPrice:    dbOrder.ProductGstPrice,
			ProductDiscount:    dbOrder.ProductDiscount,
			Quantity:           dbOrder.Quantity,
			OrderNumber:        dbOrder.OrderNumber,
			TotalPrice:         dbOrder.TotalPrice,
			PaymentStatus:      dbOrder.PaymentStatus,
			OrderStatus:        dbOrder.OrderStatus,
			BarcodeCount:       dbOrder.BarcodeCount,
			BarcodeNumbers:     dbOrder.BarcodeNumbers,
			TrackingID:         dbOrder.TrackingID,
			CreatedAt:          dbOrder.CreatedAt,
			Customer:           customer,
			Payments:           payments,
		})
	}
	response := OrdersFetchedDataResponse{
		Page:          page,
		PerPage:       perPage,
		Sort:          sort,
		SortColumn:    sortColumn,
		SearchText:    searchText,
		TotalRecords:  totalRecords,
		TotalPages:    totalPages,
		PaymentStatus: paymentStatus,
		OrderStatus:   orderStatus,
		Records:       responseOrders,
	}

	utils.JSONResponse(c, http.StatusOK, utils.MsgOrdersFetchedSuccessfully, response)
}

// AssignKit handles assigning multiple barcodes to an order
func AssignKit(c *gin.Context) {
	// Get user ID from context
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Parse and Validate Request Body
	var req CreateAssignKitRequest
	allowedFields := []string{"barcode_numbers"}
	if err := utils.ParseRequestBody(c.Request, &req, allowedFields); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Split the barcode numbers and trim spaces
	barcodeNumbers := strings.Split(strings.TrimSpace(req.BarcodeNumbers), ",")

	// Remove empty strings from the slice
	var validBarcodes []string
	for _, code := range barcodeNumbers {
		if trimmed := strings.TrimSpace(code); trimmed != "" {
			validBarcodes = append(validBarcodes, trimmed)
		}
	}
	if len(validBarcodes) == 0 {
		utils.JSONResponse(c, http.StatusBadRequest, "No valid barcodes provided", nil)
		return
	}

	// Validate each barcode
	for _, barcode := range validBarcodes {
		// Check length
		if len(barcode) != 30 {
			utils.JSONResponse(c, http.StatusBadRequest,
				fmt.Sprintf("Invalid QRCode/Barcode length for %s: must be 30 characters", barcode), nil)
			return
		}

		// Validate format (either alphabetic or numeric, no spaces)
		isValidBarcode := regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(barcode) || regexp.MustCompile(`^\d+$`).MatchString(barcode)

		if !isValidBarcode {
			utils.JSONResponse(c, http.StatusBadRequest,
				fmt.Sprintf("Invalid QRCode/Barcode format for %s: must be either alphabetic or numeric, and no spaces", barcode), nil)
			return
		}
	}

	// Extract Order ID from URL
	orderID := c.Param("id")

	// Begin Transaction
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
	}()

	// Fetch current user details
	var currentUser models.User
	if err := tx.Where("id = ? AND is_deleted = ?", user.ID, false).First(&currentUser).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Check if the order exists and is not deleted
	var existingOrder models.Order
	if err := tx.Where("id = ? AND is_deleted = ?", orderID, false).First(&existingOrder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusNotFound, utils.MsgOrderNotFound, nil)
			return
		}
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Check if the order's payment status is "Completed"
	if existingOrder.PaymentStatus != "Completed" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgPaymentnotCompleted, nil)
		return
	}

	// Count existing barcodes for this order
	var existingBarcodeCount int64
	if err := tx.Model(&models.Barcode{}).Where("order_id = ?", orderID).Count(&existingBarcodeCount).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Check if total barcodes would exceed order quantity
	if int(existingBarcodeCount)+len(validBarcodes) > int(existingOrder.Quantity) {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, "Total number of barcodes would exceed order quantity", nil)
		return
	}

	// Check if any of the barcodes are already assigned
	var existingBarcode models.Barcode
	for _, barcode := range validBarcodes {
		if err := tx.Where("barcode_number = ?", barcode).First(&existingBarcode).Error; err == nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusConflict,
				fmt.Sprintf("QRCode/Barcode %s is already assigned", barcode), nil)
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
			return
		}
	}

	// Convert orderID (string) to uint
	orderIDUint, err := strconv.ParseUint(orderID, 10, 32)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidOrderId, nil)
		return
	}

	// Create new Barcode entries
	for _, barcode := range validBarcodes {
		newBarcode := models.Barcode{
			BarcodeNumber: barcode,
			OrderID:       uint(orderIDUint),
		}
		if err := tx.Create(&newBarcode).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
			return
		}
	}

	// If barcodes match order quantity, update order status
	if int(existingBarcodeCount)+len(validBarcodes) == int(existingOrder.Quantity) {
		// Retrieve existing status history
		var statusHistory []models.StatusEntry
		if len(existingOrder.OrderStatusHistory) > 0 {
			if err := json.Unmarshal(existingOrder.OrderStatusHistory, &statusHistory); err != nil {
				tx.Rollback()
				utils.JSONResponse(c, http.StatusInternalServerError, "Failed to parse status history", nil)
				return
			}
		}

		// Add new status entry
		newStatusEntry := models.StatusEntry{
			Status:    "Processing",
			Timestamp: time.Now(),
		}
		statusHistory = append(statusHistory, newStatusEntry)

		// Convert status history to JSON
		statusHistoryJSON, err := json.Marshal(statusHistory)
		if err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create status history", nil)
			return
		}

		// Update order status and status history
		updateResult := tx.Model(&existingOrder).Updates(map[string]interface{}{
			"order_status":         "Processing",
			"order_status_history": statusHistoryJSON,
		})

		if updateResult.Error != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
			return
		}

		// Fetch customer details
		var customer models.Customer
		if err := tx.First(&customer, "id = ?", existingOrder.CustomerID).Error; err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
			return
		}

		// Send Email Notification to customer
		if err := sendOrderStatusChangeEmail(&customer, &existingOrder); err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
			return
		}
	}
	notificationUsername := fmt.Sprintf(
		"%s %s",
		currentUser.FirstName, currentUser.LastName,
	)

	// Send notification to super-admin and admin
	message := fmt.Sprintf(
		"%s %s has assigned a kit. Order ID: %d, Barcodes: %s",
		currentUser.FirstName,
		currentUser.LastName,
		existingOrder.ID,
		strings.Join(validBarcodes, ", "),
	)
	metadata := map[string]interface{}{
		"status":       existingOrder.OrderStatus,
		"module":       "Manage Orders",
		"order_number": existingOrder.OrderNumber,
	}
	err = services.NotifyUsersByRoles(
		tx,
		nil,
		[]string{"super-admin", "admin"},
		"Kits Assigned",
		"Order Management",
		message,
		notificationUsername,
		"orders",
		existingOrder.ID,
		metadata,
	)

	if err != nil {
	}

	// Commit Transaction
	tx.Commit()

	// Send Success Response
	utils.JSONResponse(c, http.StatusOK, fmt.Sprintf("%d barcodes assigned successfully", len(validBarcodes)), nil)
}

// UpdateOrderStatus updates the status of an order
func UpdateOrderStatus(c *gin.Context) {
	// Get user ID from context
	user, ok := middlewares.GetUserFromContext(c)
	if !ok {
		utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
		return
	}

	// Parse and Validate Request Body
	var req UpdateOrderStatusRequest
	allowedFields := []string{"status", "tracking_id"}
	if err := utils.ParseRequestBody(c.Request, &req, allowedFields); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Trim and Normalize the Status
	req.Status = strings.TrimSpace(req.Status)
	newStatus := strings.Title(strings.ToLower(req.Status))
	req.TrackingID = strings.TrimSpace(req.TrackingID)

	// Validate the status
	if !utils.StringInSlice(strings.ToLower(newStatus), toLowerSlice(allowedStatuses)) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidOrderStatusReq, nil)
		return
	}

	// Validate TrackingID (optional)
	if len(req.TrackingID) > 50 {
		utils.JSONResponse(c, http.StatusBadRequest, "Tracking ID must be less than or equal to 50 characters", nil)
		return
	}

	// Get the order ID from the request parameters
	orderID := c.Param("id")
	if orderID == "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgOrderIdRequired, nil)
		return
	}

	// Begin Transaction
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
	}()

	// Fetch current user details
	var currentUser models.User
	if err := tx.Where("id = ? AND is_deleted = ?", user.ID, false).First(&currentUser).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
			return
		}
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Fetch the order
	var order models.Order
	if err := tx.First(&order, "id = ? AND is_deleted = ?", orderID, false).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusNotFound, utils.MsgOrderNotFound, nil)
		} else {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
		return
	}

	// Ensure case-insensitive comparison for downgrade or same status
	currentStatus := strings.ToLower(order.OrderStatus)
	if strings.ToLower(newStatus) == currentStatus {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgOrderAlreadySetToSameValue, nil)
		return
	}

	// Check if payment is completed
	if order.PaymentStatus != "Completed" {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgPaymentnotCompleted, nil)
		return
	}

	// Check if the order has a barcode assigned
	var barcode models.Barcode
	if err := tx.First(&barcode, "order_id = ?", order.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgBarcodeNotAssigned, nil)
		} else {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		}
		return
	}

	// Retrieve existing status history
	var statusHistory []models.StatusEntry
	if len(order.OrderStatusHistory) > 0 {
		if err := json.Unmarshal(order.OrderStatusHistory, &statusHistory); err != nil {
			tx.Rollback()
			utils.JSONResponse(c, http.StatusInternalServerError, "Failed to parse status history", nil)
			return
		}
	}

	// Add new status entry
	newStatusEntry := models.StatusEntry{
		Status:    newStatus,
		Timestamp: time.Now(),
	}
	statusHistory = append(statusHistory, newStatusEntry)

	// Convert status history to JSON
	statusHistoryJSON, err := json.Marshal(statusHistory)
	if err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to create status history", nil)
		return
	}

	// Update the order status and status history
	updateResult := tx.Model(&order).Updates(map[string]interface{}{
		"order_status":         newStatus,
		"order_status_history": statusHistoryJSON,
		"tracking_id":          req.TrackingID,
	})

	if updateResult.Error != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Fetch customer details
	var customer models.Customer
	if err := tx.First(&customer, "id = ?", order.CustomerID).Error; err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Send Email Notification to customer
	if err := sendOrderStatusChangeEmail(&customer, &order); err != nil {
		tx.Rollback()
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}
	notificationUsername := fmt.Sprintf(
		"%s %s",
		currentUser.FirstName, currentUser.LastName,
	)

	// Send notification to super-admin and admin
	message := fmt.Sprintf(
		"Order status updated by %s %s. Order ID: %d, New Status: %s",
		currentUser.FirstName,
		currentUser.LastName,
		order.ID,
		newStatus,
	)
	metadata := map[string]interface{}{
		"status":       newStatus,
		"module":       "Manage Orders",
		"order_number": order.OrderNumber,
	}
	err = services.NotifyUsersByRoles(
		tx,
		nil,
		[]string{"super-admin", "admin"},
		"Order Status Updated",
		"Order Management",
		message,
		notificationUsername,
		"orders",
		order.ID,
		metadata,
	)

	if err != nil {
	}

	// Commit Transaction
	tx.Commit()

	utils.JSONResponse(c, http.StatusOK, utils.MsgOrderStatusUpdateSuccessfully, nil)
}

// Helper function to convert a slice of strings to lowercase
func toLowerSlice(slice []string) []string {
	lowerSlice := make([]string, len(slice))
	for i, v := range slice {
		lowerSlice[i] = strings.ToLower(v)
	}
	return lowerSlice
}

func sendOrderStatusChangeEmail(customer *models.Customer, order *models.Order) error {
	emailBody := emails.OrderStatusUpdateEmail(
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
		order.TrackingID,
		customer.ClinicID,
		order.ProductName,
		order.ProductDescription,
		order.Quantity,
		order.ProductPrice,
		order.ProductGstPrice,
		order.ProductDiscount,
		order.TotalPrice,
		order.OrderStatus,
	)

	// Send email to the customer
	if err := config.SendEmail([]string{customer.Email}, "Order Status Update", emailBody); err != nil {
		return err
	}

	return nil
}

// GetOrderCounts fetches counts of orders based on their statuses and payment status
func GetOrderCounts(c *gin.Context) {
	// Slice to hold the grouped results
	var orderCountResults []struct {
		OrderStatus string
		Count       int64
	}

	// Variable to store total quantity
	var totalQuantity int64

	// First query: Get order counts grouped by status
	result := config.DB.Model(&models.Order{}).
		Select("order_status, COUNT(*) as count").
		Where("payment_status = ? AND is_deleted = ?", "Completed", false).
		Group("order_status").
		Scan(&orderCountResults)

	if result.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Error fetching order counts", nil)
		return
	}

	// Second query: Get total quantity
	result = config.DB.Model(&models.Order{}).
		Select("COALESCE(SUM(quantity), 0) as total_quantity").
		Where("payment_status = ? AND is_deleted = ?", "Completed", false).
		Scan(&totalQuantity)

	if result.Error != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Error fetching total quantity", nil)
		return
	}

	// Initialize response with zero values
	response := OrderCountResponse{
		Dispatched:    0,
		Processing:    0,
		Pending:       0,
		Shipped:       0,
		TotalQuantity: totalQuantity,
	}

	// Map the results to the response structure
	for _, item := range orderCountResults {
		switch item.OrderStatus {
		case "Dispatched":
			response.Dispatched = item.Count
		case "Processing":
			response.Processing = item.Count
		case "Pending":
			response.Pending = item.Count
		case "Shipped":
			response.Shipped = item.Count
		}
	}

	// Send the JSON response
	utils.JSONResponse(c, http.StatusOK, utils.MsgOrderCountFetchedSuccessfully, response)
}
