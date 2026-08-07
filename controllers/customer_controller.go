// customer_controller.go
package controllers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/utils"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OrderDetails represents the structure for order information
type OrderDetails struct {
	OrderID         uint    `json:"order_id"`
	ProductName     string  `json:"product_name"`
	TotalPrice      float64 `json:"total_price"`
	PaymentStatus   string  `json:"payment_status"`
	Quantity        int     `json:"quantity"`
	OrderNumber     string  `json:"order_number"`
	ProductDiscount float64 `json:"product_discount"`
	ProductGstPrice float64 `json:"product_gst_price"`
	TrackingID      string  `json:"tracking_id"`
}

// CustomerWithOrders represents a customer with all their orders
type CustomerWithOrders struct {
	ID            uint           `json:"id"`
	FirstName     string         `json:"first_name"`
	LastName      string         `json:"last_name"`
	Email         string         `json:"email"`
	PhoneNumber   string         `json:"phone_number"`
	Country       string         `json:"country"`
	StreetAddress string         `json:"street_address"`
	TownCity      string         `json:"town_city"`
	Region        string         `json:"region"`
	Postcode      string         `json:"postcode"`
	ClinicID      string         `json:"clinic_id"`
	CreatedAt     time.Time      `json:"created_at"`
	Orders        []OrderDetails `json:"orders"`
}

// Response structure for paginated list of customers with their orders
type CustomerFetchedDataResponse struct {
	Page         int                  `json:"page"`
	PerPage      int                  `json:"per_page"`
	Sort         string               `json:"sort"`
	SortColumn   string               `json:"sort_column"`
	SearchText   string               `json:"search_text"`
	TotalRecords int64                `json:"total_records"`
	TotalPages   int                  `json:"total_pages"`
	Records      []CustomerWithOrders `json:"records"`
}

// Response structure for paginated list of customers with their orders
type SpecificCustomerFetchedDataResponse struct {
	Page         int                      `json:"page"`
	PerPage      int                      `json:"per_page"`
	Sort         string                   `json:"sort"`
	SortColumn   string                   `json:"sort_column"`
	SearchText   string                   `json:"search_text"`
	TotalRecords int64                    `json:"total_records"`
	TotalPages   int                      `json:"total_pages"`
	Records      []CustomerOrdersResponse `json:"records"`
}

// CustomerOrdersResponse defines the structure for the API response
type CustomerOrdersResponse struct {
	ID                 uint                           `json:"id"`
	ProductName        string                         `json:"product_name"`
	ProductDescription string                         `json:"product_description"`
	ProductImage       string                         `json:"product_image"`
	ProductPrice       float64                        `json:"product_price"`
	ProductGstPrice    float64                        `json:"product_gst_price"`
	Quantity           int                            `json:"quantity"`
	TotalPrice         float64                        `json:"total_price"`
	PaymentStatus      string                         `json:"payment_status"`
	OrderStatus        string                         `json:"order_status"`
	OrderNumber        string                         `json:"order_number"`
	ProductDiscount    float64                        `json:"product_discount"`
	TrackingID         string                         `json:"tracking_id"`
	CreatedAt          time.Time                      `json:"created_at"`
	Barcodes           []BarcodeResponse              `json:"barcodes"` // New field
	Payments           []CustomerOrderPaymentResponse `json:"payments,omitempty"`
}

// Add new BarcodeResponse struct
type BarcodeResponse struct {
	ID            uint      `json:"id"`
	BarcodeNumber string    `json:"barcode_number"`
	CreatedAt     time.Time `json:"created_at"`
}

// Customer represents the essential customer information
type SpecificCustomerResponse struct {
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
	ShippingCountry  string    `json:"shipping_country"`
	ShippingAddress  string    `json:"shipping_address"`
	ShippingTownCity string    `json:"shipping_town_city"`
	ShippingRegion   string    `json:"shipping_region"`
	ShippingPostcode string    `json:"shipping_postcode"`
	CreatedAt        time.Time `json:"created_at"`
}

// Payment represents the payment information with related invoices
type CustomerOrderPaymentResponse struct {
	ID              uint                          `json:"id"`
	PaymentStatus   string                        `json:"payment_status"`
	TransactionID   string                        `json:"transaction_id"`
	Amount          float64                       `json:"amount"`
	ProductGstPrice float64                       `json:"product_gst_price"`
	CreatedAt       time.Time                     `json:"created_at"`
	Invoices        []CustomerOrderPaymentInvoice `json:"invoices,omitempty"`
}

// Invoice represents the invoice information
type CustomerOrderPaymentInvoice struct {
	ID              uint    `json:"id"`
	InvoiceLink     string  `json:"invoice_link"`
	Price           float64 `json:"price"`
	ProductGstPrice float64 `json:"product_gst_price"`
	InvoiceID       string  `json:"invoice_id"`
}

// GetCustomersWithOrders handles fetching customers with their orders
func GetCustomersWithOrders(c *gin.Context) {
	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text"}

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
	validSortColumns := []string{"first_name", "last_name", "email", "phone_number", "country", "street_address", "town_city", "region", "postcode", "created_at", "clinic_id"}
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

	// Initialize query for fetching customers
	var customers []models.Customer
	customersDB := config.DB.Debug().
		Model(&models.Customer{}).
		Where("customers.is_deleted = ?", false)

	// Apply search filter if provided
	if searchText != "" {
		searchPattern := "%" + searchText + "%"
		customersDB = customersDB.Where(
			"customers.first_name ILIKE ? OR customers.last_name ILIKE ? OR customers.clinic_id ILIKE ? OR customers.email ILIKE ? OR customers.phone_number ILIKE ? OR customers.country ILIKE ? OR customers.street_address ILIKE ? OR customers.town_city ILIKE ? OR customers.region ILIKE ? OR customers.postcode ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Get total records count
	var totalRecords int64
	if err := customersDB.Count(&totalRecords).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgFailedToCountRecords, nil)
		return
	}

	// Calculate total pages
	totalPages := int((totalRecords + int64(perPage) - 1) / int64(perPage))
	if totalRecords == 0 {
		totalPages = 0
	}

	// Apply sorting and pagination
	customersDB = customersDB.Order(sortColumn + " " + sort)
	offset := (page - 1) * perPage
	if err := customersDB.Limit(perPage).Offset(offset).Find(&customers).Error; err != nil {
		log.Printf("Error fetching customers: %v", err)
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Prepare the response with customers and their orders
	var customerWithOrders []CustomerWithOrders

	// For each customer, fetch their orders
	for _, customer := range customers {
		var orders []models.Order
		if err := config.DB.Debug().
			Model(&models.Order{}).
			Where("customer_id = ? AND is_deleted = ?", customer.ID, false).
			Find(&orders).Error; err != nil {
			log.Printf("Error fetching orders for customer %d: %v", customer.ID, err)
			continue
		}

		// Convert orders to OrderDetails
		var orderDetails []OrderDetails
		for _, order := range orders {
			orderDetails = append(orderDetails, OrderDetails{
				OrderID:         order.ID,
				ProductName:     order.ProductName,
				TotalPrice:      order.TotalPrice,
				PaymentStatus:   order.PaymentStatus,
				Quantity:        order.Quantity,
				OrderNumber:     order.OrderNumber,
				ProductGstPrice: order.ProductGstPrice,
				ProductDiscount: order.ProductDiscount,
				TrackingID:      order.TrackingID,
			})
		}

		// Add customer with their orders to the result
		customerWithOrders = append(customerWithOrders, CustomerWithOrders{
			ID:            customer.ID,
			FirstName:     customer.FirstName,
			LastName:      customer.LastName,
			Email:         customer.Email,
			PhoneNumber:   customer.PhoneNumber,
			Country:       customer.Country,
			StreetAddress: customer.StreetAddress,
			TownCity:      customer.TownCity,
			Region:        customer.Region,
			Postcode:      customer.Postcode,
			ClinicID:      customer.ClinicID,
			CreatedAt:     customer.CreatedAt,
			Orders:        orderDetails,
		})
	}

	// Prepare response
	response := CustomerFetchedDataResponse{
		Page:         page,
		PerPage:      perPage,
		Sort:         sort,
		SortColumn:   sortColumn,
		SearchText:   searchText,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Records:      customerWithOrders,
	}

	// Send response
	utils.JSONResponse(c, http.StatusOK, utils.MsgCustomerFetchedSuccessfully, response)
}

func GetCustomerOrderDetails(c *gin.Context) {
	// Extract customer ID from URL parameters
	customerID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidCustomerId, nil)
		return
	}

	// Define allowed query parameters
	allowedFields := []string{"page", "per_page", "sort", "sort_column", "search_text", "payment_status", "order_status"}

	// Parse query parameters with default values
	query := c.Request.URL.Query()
	if !utils.AllowFields(query, allowedFields) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidQueryParameters, nil)
		return
	}
	// Pagination parameters
	page := 1
	if val := query.Get("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPageParameter, nil)
			return
		}
	}

	perPage := 10
	if val := query.Get("per_page"); val != "" {
		if pp, err := strconv.Atoi(val); err == nil && pp > 0 {
			perPage = pp
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPerPageParameter, nil)
			return
		}
	}

	// Sorting parameters
	sort := "desc"
	if val := strings.ToLower(query.Get("sort")); val == "asc" || val == "desc" {
		sort = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortParameter, nil)
		return
	}

	// Sort column validation
	sortColumn := "created_at"
	validSortColumns := []string{
		"id", "product_name", "product_price", "product_gst_price",
		"quantity", "total_price", "payment_status",
		"order_status", "created_at", "order_number",
	}
	if val := strings.ToLower(query.Get("sort_column")); val != "" {
		if utils.StringInSlice(val, validSortColumns) {
			sortColumn = val
		} else {
			utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidSortColumnParameter, nil)
			return
		}
	}

	// Search text
	searchText := strings.TrimSpace(query.Get("search_text"))

	// Payment status filter
	paymentStatus := "all"
	if val := strings.ToLower(query.Get("payment_status")); val == "pending" || val == "completed" || val == "failed" || val == "all" {
		paymentStatus = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidPaymentStatus, nil)
		return
	}

	// Order status filter
	orderStatus := "all"
	if val := strings.ToLower(query.Get("order_status")); val == "pending" || val == "shipped" || val == "delivered" || val == "processing" || val == "dispatched" || val == "cancelled" || val == "all" {
		orderStatus = val
	} else if val != "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidOrderStatus, nil)
		return
	}

	// First, verify customer exists
	var customer models.Customer
	if err := config.DB.First(&customer, customerID).Error; err != nil {
		utils.JSONResponse(c, http.StatusNotFound, utils.MsgCustomerNotFound, nil)
		return
	}

	// Prepare orders query
	ordersDB := config.DB.Model(&models.Order{}).
		Where("customer_id = ? AND is_deleted = ?", customerID, false)

	// Apply search filter
	if searchText != "" {
		searchPattern := "%" + searchText + "%"
		ordersDB = ordersDB.Where(
			`product_name ILIKE ? OR 
			CAST(product_price AS TEXT) ILIKE ? OR 
			CAST(product_gst_price AS TEXT) ILIKE ? OR 
			CAST(quantity AS TEXT) ILIKE ? OR 
			CAST(total_price AS TEXT) ILIKE ? OR 
			CAST(order_number AS TEXT) ILIKE ? OR
			payment_status ILIKE ? OR 
			order_status ILIKE ? OR 
			TO_CHAR(created_at, 'YYYY-MM-DD') ILIKE ?`,
			searchPattern, searchPattern, searchPattern, searchPattern,
			searchPattern, searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Apply payment status filter
	if paymentStatus != "all" {
		ordersDB = ordersDB.Where("LOWER(payment_status) = ?", strings.ToLower(paymentStatus))
	}

	// Apply order status filter
	if orderStatus != "all" {
		ordersDB = ordersDB.Where("LOWER(order_status) = ?", strings.ToLower(orderStatus))
	}

	// Count total records
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

	// Apply sorting
	ordersDB = ordersDB.Order(sortColumn + " " + sort)

	// Apply pagination
	offset := (page - 1) * perPage
	var orders []models.Order
	if err := ordersDB.
		Preload("Payments", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_deleted = ?", false)
		}).
		Preload("Payments.Invoices", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_deleted = ?", false)
		}).
		Preload("Barcodes", func(db *gorm.DB) *gorm.DB { // Add this line
			return db.Where("is_deleted = ?", false)
		}).
		Limit(perPage).
		Offset(offset).
		Find(&orders).Error; err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgInternalServerError, nil)
		return
	}

	// Transform orders to response format
	var responseOrders []CustomerOrdersResponse
	for _, dbOrder := range orders {
		var payments []CustomerOrderPaymentResponse
		for _, dbPayment := range dbOrder.Payments {
			var invoices []CustomerOrderPaymentInvoice
			for _, dbInvoice := range dbPayment.Invoices {
				invoices = append(invoices, CustomerOrderPaymentInvoice{
					ID:              dbInvoice.ID,
					InvoiceLink:     dbInvoice.InvoiceLink,
					Price:           dbInvoice.Price,
					ProductGstPrice: dbInvoice.ProductGstPrice,
					InvoiceID:       dbInvoice.InvoiceID,
				})
			}
			payments = append(payments, CustomerOrderPaymentResponse{
				ID:              dbPayment.ID,
				PaymentStatus:   dbPayment.PaymentStatus,
				TransactionID:   dbPayment.TransactionID,
				Amount:          dbPayment.Amount,
				ProductGstPrice: dbPayment.ProductGstPrice,
				CreatedAt:       dbPayment.CreatedAt,
				Invoices:        invoices,
			})
		}

		// Transform barcodes
		var barcodes []BarcodeResponse
		for _, dbBarcode := range dbOrder.Barcodes {
			barcodes = append(barcodes, BarcodeResponse{
				ID:            dbBarcode.ID,
				BarcodeNumber: dbBarcode.BarcodeNumber,
				CreatedAt:     dbBarcode.CreatedAt,
			})
		}

		responseOrders = append(responseOrders, CustomerOrdersResponse{
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
			TrackingID:         dbOrder.TrackingID,
			CreatedAt:          dbOrder.CreatedAt,
			Payments:           payments,
			Barcodes:           barcodes,
		})
	}

	// Prepare customer response
	customerResponse := SpecificCustomerResponse{
		ID:               customer.ID,
		FirstName:        customer.FirstName,
		LastName:         customer.LastName,
		Email:            customer.Email,
		PhoneNumber:      customer.PhoneNumber,
		Country:          customer.Country,
		StreetAddress:    customer.StreetAddress,
		TownCity:         customer.TownCity,
		Region:           customer.Region,
		Postcode:         customer.Postcode,
		ClinicID:         customer.ClinicID,
		CreatedAt:        customer.CreatedAt,
		ShippingCountry:  customer.ShippingCountry,
		ShippingAddress:  customer.ShippingAddress,
		ShippingTownCity: customer.ShippingTownCity,
		ShippingRegion:   customer.ShippingRegion,
		ShippingPostcode: customer.ShippingPostcode,
	}

	// Construct final response
	response := struct {
		Customer SpecificCustomerResponse            `json:"customer"`
		Orders   SpecificCustomerFetchedDataResponse `json:"orders"`
	}{
		Customer: customerResponse,
		Orders: SpecificCustomerFetchedDataResponse{
			Page:         page,
			PerPage:      perPage,
			Sort:         sort,
			SortColumn:   sortColumn,
			SearchText:   searchText,
			TotalRecords: totalRecords,
			TotalPages:   totalPages,
			Records:      responseOrders,
		},
	}

	// Send response
	utils.JSONResponse(c, http.StatusOK, utils.MsgCustomerFetchedSuccessfully, response)
}
